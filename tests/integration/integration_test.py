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


def _AsyncStreamPrint(output_stream, stream_name):
    """Performs reads on input stream and prints them without blocking.

    Args:
      output_stream: Stream of output to print asynchronously (typically a
          subprocess.PIPE).
      stream_name: Display name of the stream to print before each line read.

    Returns:
      threading.Event() that will halt the thread (after the next stream read)
          when set.
    """
    cancel_event = threading.Event()
    reader_thread = threading.Thread(
        target=_PrintOutputLines,
        args=(output_stream, stream_name, cancel_event))
    reader_thread.daemon = True
    reader_thread.start()
    return cancel_event


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


def _DCPInfrastructureCommand(command, insert_job=False):
    """Runs the infrastructure command to setup/teardown DCP infrastructure.

    Args:
      command: Either 'Create', 'TearDown' or 'CreateThenTearDown'.
      insert_job: Whether to insert a new job to the system.

    Raises:
      Exception: If the command failed for any reason.
    """
    # TODO: Migrate away from personal docker package (here and below).
    # TODO: Presently, this creates a new GCE VM (which takes a while to
    # start up). When the DCP setup script supports local deployment,
    # do that instead to improve performance.
    create_infra_command = [
        'sudo', 'docker', 'run', '-it' if _IsRunningInteractively() else '-i',
        'gcr.io/mbassiouny-test/cloud-ingest:dcp',
        'python', 'create-infra/main.py',
        command  # Create or tear down infrastructure.
    ]
    if insert_job:
        create_infra_command.extend([
            '-j',  # Insert new job
            '--src-dir', TMP_DIR,
            '--dst-gcs-bucket', _BUCKET_NAME,
            '--dst-bq-datase', _DATASET_NAME, '--dst-bq-table', _TABLE_NAME
        ])
    print 'Infrastructure command:\n%s' % (
        ' '.join(create_infra_command))
    create_infra_process = subprocess.Popen(
        create_infra_command,
        stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    infra_stdout_stop = _AsyncStreamPrint(
        create_infra_process.stdout, 'infra stdout')
    infra_stderr_stop = _AsyncStreamPrint(
        create_infra_process.stderr, 'infra stderr')
    return_code = create_infra_process.wait()
    infra_stdout_stop.set()
    infra_stderr_stop.set()

    if return_code != 0:
        raise Exception('DCP infrastructure command failed with '
                        'return code %d' % return_code)


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
    print 'Starting gsutil.  Command line:\n%s\n' % ' '.join(gsutil_command)
    gsutil_process = subprocess.Popen(
         gsutil_command,
         stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    stdout_stop = _AsyncStreamPrint(gsutil_process.stdout,
        'gsutil stdout')
    stderr_stop = _AsyncStreamPrint(gsutil_process.stderr,
        'gsutil stderr')
    return stdout_stop, stderr_stop



if __name__ == "__main__":
    # TODO: Presently, this test runs from GCE and expects service account
    # credentials on the GCE instance with cloud-platform auth scope. Extend
    # this to alternatively accept a service account credential key file.

    PROJECT_ID = _STORAGE_CLIENT.project

    # Clean up old infrastructure, if they exist.
    _DCPInfrastructureCommand('TearDown')

    try:
        _CreateBQDatasetAndTable()

        # Set up the GCS staging bucket for objects to be loaded into BigQuery.
        _STORAGE_CLIENT.bucket(_BUCKET_NAME).create()

        TMP_DIR, FILE_PATHS = _CreateTempFiles()

        _DCPInfrastructureCommand('Create', insert_job=True)
        gsutil_stdout_stop, gsutil_stderr_stop = _RunGsutilAgent(PROJECT_ID)

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
        gsutil_stdout_stop.set()
        gsutil_stderr_stop.set()

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
        _DCPInfrastructureCommand('TearDown')
