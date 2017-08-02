# -*- coding: utf-8 -*-
# Copyright 2017 Google Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License

"""Integration test for on-premises ingest into BigQuery via GCS."""
import argparse
import os
import random
import string
import subprocess
import sys
import tempfile
import threading
import time

# pylint: disable=import-error,no-name-in-module,no-member,invalid-name
import google.cloud
from google.cloud import bigquery
from google.cloud import storage

_TEST_SUFFIX_LENGTH = 6
_TEST_SUFFIX = ''.join(
    random.choice(string.ascii_lowercase + string.digits)
    for _ in xrange(_TEST_SUFFIX_LENGTH))

_DATASET_NAME = 'opi_integration_test_dataset%s' % _TEST_SUFFIX
_TABLE_NAME = 'opi_integration_test_table%s' % _TEST_SUFFIX
_BUCKET_NAME = 'opi-integration-test-bucket%s' % _TEST_SUFFIX
_LIST_OUTPUT_OBJECT_NAME = 'opi-integration-test-list-output-object'

_BIGQUERY_CLIENT = bigquery.Client()
_STORAGE_CLIENT = storage.Client()

# Source data for the test; 3 files which should eventually result in
# 15 BigQuery rows.
_FILE_CONTENTS = (
    'Ann,5\nBetsy,8\nCarlos,6\nDmitri,24',
    'Elizabeth,12\nFrank,14\nGarrett,75\nHelen,35\nIvan,12',
    'James,2\nKelson,67\nLorax,53\nMadeline,22\nNorn,11\nOpi,1\n',
)
# One object per source file, plus one list output object
_NUM_EXPECTED_OBJECTS = len(_FILE_CONTENTS) + 1
_NUM_EXPECTED_BQ_ROWS = sum(len(contents.split())
                            for contents in _FILE_CONTENTS)


def _CleanUpBq(bq_client):
    """Deletes the existing BigQuery dataset/table, if they exist."""
    try:
        bq_dataset = bq_client.dataset(_DATASET_NAME)
        bq_table = bq_dataset.table(_TABLE_NAME)
        bq_table.delete()
        bq_dataset.delete()
    except google.cloud.exceptions.NotFound:
        pass


def _CleanUpGCS(storage_client):
    """Deletes the GCS bucket and contained blobs, if they exist."""
    try:
        for blob in storage_client.bucket(_BUCKET_NAME).list_blobs():
            blob.delete()
    except google.cloud.exceptions.NotFound:
        pass
    try:
        storage_client.bucket(_BUCKET_NAME).delete()
    except google.cloud.exceptions.NotFound:
        pass


def _CreateTempFiles():
    """Creates temporary CSV files for eventual insertion into BigQuery."""
    tmp_dir = tempfile.mkdtemp()
    tmp_file_paths = []
    for i in xrange(3):
        file_name = 'file%s' % i
        fpath = os.path.join(tmp_dir, file_name)
        tmp_file_paths.append(fpath)
        with open(fpath, 'wb') as file_pointer:
            file_pointer.write(_FILE_CONTENTS[i])
    return tmp_dir, tmp_file_paths

def _WaitForFunc(input_func, timeout=120):
    """Runs input_func until successful or timeout (seconds) is reached."""
    start_time = time.time()
    while not input_func():
        if time.time() - start_time > timeout:
            raise Exception(
                'Input function %s did not complete within %s seconds.' %
                (input_func, timeout))
        time.sleep(1)


def _GetNumGCSObjects():
    """Returns the number of GCS objects in the staging bucket."""
    bucket = _STORAGE_CLIENT.bucket(_BUCKET_NAME)
    return len(list(bucket.list_blobs()))


def _GetNumBQRows():
    """Returns the number of BigQuery rows in the table."""
    bq_dataset = _BIGQUERY_CLIENT.dataset(_DATASET_NAME)
    bq_table = bq_dataset.table(_TABLE_NAME)
    bq_table.reload()
    num_rows = bq_table.num_rows
    if num_rows > 0:
        print 'Found %s rows in BigQuery table %s' % (num_rows, _TABLE_NAME)
    return num_rows


def _PrintOutputLines(output_stream, stream_name, cancel_event):
    """Prints lines from output_stream until cancel_event is set."""
    for line in iter(output_stream.readline, b''):
        if cancel_event.isSet():
            return
        print '%s: %s' % (stream_name, line.rstrip('\n'))


class _AsyncStreamPrinter(object):
    """Performs reads on input stream and prints them without blocking.

    This class prints the stream as it is written, and also buffers the
    stream locally in self.buffered_stream_output so that the contents can
    be read and compared.

    To stop printing/buffering, call self.cancel_event.set().
    """

    def __init__(self, output_stream, stream_name):
        """Initializes the printer.

        Args:
          output_stream: Stream of output to print asynchronously (typically a
              subprocess.PIPE).
          stream_name: Display name of the stream to accompany each printout.
        """
        self._cancel_event = threading.Event()
        self._buffered_stream_output = b''

        reader_thread = threading.Thread(
            target=self._PrintOutputLines,
            args=(output_stream, stream_name, self.cancel_event))
        reader_thread.daemon = True
        reader_thread.start()

    @property
    def cancel_event(self):
        """Getter for cancel_event."""
        return self._cancel_event

    @property
    def buffered_stream_output(self):
        """Getter for buffered_stream_output."""
        return self._buffered_stream_output

    def _PrintOutputLines(self, output_stream, stream_name, cancel_event):
        """Prints lines from output_stream until cancel_event is set."""
        for line in iter(output_stream.readline, b''):
            self._buffered_stream_output += line
            if cancel_event.isSet():
                return
            print '%s: %s' % (stream_name, line.rstrip('\n'))


def _CreateBQDatasetAndTable():
    """Creates the (empty) test BigQuery dataset and table."""
    dataset = _BIGQUERY_CLIENT.dataset(_DATASET_NAME)
    dataset.create()
    print 'Dataset {} created.'.format(dataset.name)
    table = dataset.table(_TABLE_NAME)
    table.schema = (
        bigquery.SchemaField('Name', 'STRING'),
        bigquery.SchemaField('Age', 'INTEGER'),
    )
    table.create()
    print 'Table {} created.'.format(table.name)


def _IsRunningInteractively():
    """Returns True if currently running interactively on a TTY."""
    return sys.stdout.isatty() and sys.stderr.isatty() and sys.stdin.isatty()


def _RunCommand(command, async=False, short_name=None):
    """Runs the passed command array.

    Args:
        command: String array of the command to run.
        async: Whether to run the command asynchronously.
        short_name: short name to identify the command for printing in stdout
            and stderr.

    Returns:
        tuple of _AsyncStreamPrinter for (stdout, stderr)
    """
    if not short_name:
        short_name = command[0]

    print 'Running command(%s): %s' % (short_name, ' '.join(command))

    command_process = subprocess.Popen(
        command, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    stdout_printer = _AsyncStreamPrinter(
        command_process.stdout, short_name + ' stdout')
    stderr_printer = _AsyncStreamPrinter(
        command_process.stderr, short_name + ' stderr')

    if not async:
        return_code = command_process.wait()
        stdout_printer.cancel_event.set()
        stderr_printer.cancel_event.set()

        if command_process.returncode != 0:
            raise Exception('command %s failed with return code %d' % (
                ' '.join(command), return_code))

    return stdout_printer, stderr_printer



def _PullDCPDockerImage(docker_image):
    """Pulls a fresh docker image."""
    dcp_command = ['sudo', 'docker', 'pull', docker_image]
    _RunCommand(dcp_command, short_name='pull image')


def _DCPInfrastructureCommand(command, docker_image, insert_job=False):
    """Runs the infrastructure command to setup/teardown DCP infrastructure.

    Args:
      command: Either 'Create', 'TearDown' or 'CreateThenTearDown'.
      docker_image: The DCP docker image.
      insert_job: Whether to insert a new job to the system.

    Raises:
      Exception: If the command failed for any reason.
    """
    # TODO: Migrate away from personal docker package (here and below).
    # TODO: Presently, this creates a new GCE VM (which takes a while to
    # start up). When the DCP setup script supports local deployment,
    # do that instead to improve performance.
    infra_command = [
        'sudo', 'docker', 'run', '-it' if _IsRunningInteractively() else '-i',
        docker_image,
        'python', 'create-infra/main.py',
        command,  # Create or tear down infrastructure.
        '--skip-running-dcp'
    ]
    if insert_job:
        infra_command.extend([
            '-j',  # Insert new job
            '--src-dir', TMP_DIR,
            '--dst-gcs-bucket', _BUCKET_NAME,
            '--dst-bq-dataset', _DATASET_NAME, '--dst-bq-table', _TABLE_NAME
        ])

    if command == 'TearDown':
        infra_command.append('--force')

    _RunCommand(infra_command, short_name='infra')


def _RunDCP(project_id, docker_image):
    """Runs the DCP as a container and returns the container id."""
    dcp_command = [
        'sudo', 'docker', 'run', '-d', docker_image,
        './dcpmain', project_id
    ]
    dcp_stdout_printer, _ = _RunCommand(dcp_command, short_name='Start DCP',
                                        async=False)
    return dcp_stdout_printer.buffered_stream_output.strip()


def _RunGsutilAgent(project_id):
    """Runs the gsutil worker agent locally to performing listing and copying.

    Args:
        project_id: Project ID where topics/subscriptions live.

    Returns:
        tuple of 2 threading.Events used to stop output prints.
    """
    # TODO: This currently uses topic names hard-coded by create
    # infrastructure.py. Once that is updated, update the names.
    gsutil_command = [
        os.path.join(os.path.expanduser('~'), 'gsutil/gsutil'), 'agent',
        'listandcopy',
        'projects/%s/subscriptions/cloud-ingest-list' % project_id,
        'projects/%s/topics/cloud-ingest-list-progress' % project_id,
        'projects/%s/subscriptions/cloud-ingest-copy' % project_id,
        'projects/%s/topics/cloud-ingest-copy-progress' % project_id]
    return _RunCommand(gsutil_command, async=True)


if __name__ == '__main__':
    # TODO: Presently, this test runs from GCE and expects service account
    # credentials on the GCE instance with cloud-platform auth scope. Extend
    # this to alternatively accept a service account credential key file.
    parser = argparse.ArgumentParser(
        description='Run end-end integration tests')

    parser.add_argument(
        '--docker-image', '-d', type=str,
        help='Name of docker image to run. Default is the test environment '
             'image (cloud-ingest:test). Use (cloud-ingest:dcp) to run the '
             'prod image, or use (cloud-ingest:$USER) to run your dev image.',
        default='cloud-ingest:test')
    args = parser.parse_args()

    PROJECT_ID = _STORAGE_CLIENT.project
    DCP_DOCKER_IMAGE = 'gcr.io/mbassiouny-test/%s' % args.docker_image

    # Get a fresh DCP docker image
    _PullDCPDockerImage(DCP_DOCKER_IMAGE)

    # Clean up old infrastructure, if they exist.
    _DCPInfrastructureCommand('TearDown', DCP_DOCKER_IMAGE)

    try:
        dcp_container_id = None
        _CreateBQDatasetAndTable()

        # Set up the GCS staging bucket for objects to be loaded into BigQuery.
        _STORAGE_CLIENT.bucket(_BUCKET_NAME).create()

        TMP_DIR, FILE_PATHS = _CreateTempFiles()

        _DCPInfrastructureCommand('Create', DCP_DOCKER_IMAGE, insert_job=True)
        dcp_container_id = _RunDCP(PROJECT_ID, DCP_DOCKER_IMAGE)

        gsutil_stdout_printer, gsutil_stderr_printer = (
            _RunGsutilAgent(PROJECT_ID))

        print 'Waiting for GCS objects to be created'
        _WaitForFunc(lambda: _GetNumGCSObjects() >= _NUM_EXPECTED_OBJECTS,
                     timeout=300)
        num_actual_gcs_objects = _GetNumGCSObjects()
        if num_actual_gcs_objects != _NUM_EXPECTED_OBJECTS:
            raise Exception('Expected %s GCS objects but found %s.' % (
                            _NUM_EXPECTED_OBJECTS, num_actual_gcs_objects))

        # Once the objects have been created, gsutil's work is done, but gsutil
        # will continue listening. There's no need to print its output to the
        # console any further.
        gsutil_stdout_printer.cancel_event.set()
        gsutil_stderr_printer.cancel_event.set()

        print 'Waiting for BQ rows to be created'
        _WaitForFunc(lambda: _GetNumBQRows() >= _NUM_EXPECTED_BQ_ROWS,
                     timeout=300)
        num_actual_bq_rows = _GetNumBQRows()
        if num_actual_bq_rows != _NUM_EXPECTED_BQ_ROWS:
            raise Exception('Expected %s GCS objects but found %s.' % (
                            _NUM_EXPECTED_BQ_ROWS, num_actual_bq_rows))
        # TODO: Teardown DCP infrastructure (Spanner and Pub/Sub queues)
        sys.exit(0)
    finally:
        _CleanUpBq(_BIGQUERY_CLIENT)
        _CleanUpGCS(_STORAGE_CLIENT)
        if dcp_container_id:
            _RunCommand(['sudo', 'docker', 'stop', dcp_container_id],
                        short_name='Stop DCP')
        _DCPInfrastructureCommand('TearDown', DCP_DOCKER_IMAGE)
