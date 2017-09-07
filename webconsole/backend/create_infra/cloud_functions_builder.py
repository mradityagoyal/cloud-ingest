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
# limitations under the License.

"""Google Cloud Functions admin utilities."""

import httplib
import os
import random
import re
import string
import tempfile
import time
import zipfile

import google.auth as googleauth
from google.auth.transport.requests import AuthorizedSession
from  google.cloud import exceptions
from google.cloud import storage


_RANDOM_BUCKET_CREATION_TRIALS = 5
_RANDOM_BUCKET_STRING_SIZE = 5

# Matches bucket strings of the form 'gs://bucket'
_BUCKET_OBJECT_REGEX = re.compile(r'^gs://(?P<bucket>[^/]*)/(?P<object>.*)')


def _create_source_zip(src_dir, zip_file_path):
    """Create source code zip file from src_dir."""
    with zipfile.ZipFile(zip_file_path, 'w') as zip_file:
        for root, _, src_files in os.walk(src_dir):
            for src_file in src_files:
                src_file_path = os.path.join(root, src_file)
                zip_file.write(src_file_path,
                               arcname=os.path.relpath(src_file_path, src_dir))


def _create_function_bucket(client, fn_name):
    """Creates a cloud function bucket.

    Initially, this will try to create bucket with fn_name. If the bucket
    already exists, it will try to append random suffixes until the bucket is
    created or max number of attempts has been reached.

    Args:
        client: Storage client object.
        fn_name: Cloud function name.

    Returns:
        The created bucket.

    Raises:
        Exception: If unable to create the bucket.
    """
    fn_name = '{}-{}'.format(client.project, fn_name.lower())
    bucket_name = fn_name
    for _ in range(_RANDOM_BUCKET_CREATION_TRIALS):
        try:
            bucket = client.create_bucket(bucket_name)
        except exceptions.Conflict:
            rand_suffix = ''.join(random.choice(string.ascii_lowercase)
                                  for _ in range(_RANDOM_BUCKET_STRING_SIZE))
            bucket_name = '{}-{}'.format(fn_name, rand_suffix)
        else:
            return bucket
    raise Exception(
        'Failed to create bucket for function {} after {} trails.'.format(
            fn_name, _RANDOM_BUCKET_CREATION_TRIALS))


def _delete_function_bucket(client, source_archive_url):
    """Deletes cloud function bucket.

    Deletes cloud function source code GCS object. Also removes the bucket
    containing this object if it's empty.

    Args:
        client: Storage client object.
        source_archive_url: Function source code archive GCS url.

    Raises:
        Exception: if the passed source_archive_url is invalid.
    """
    # parse out the url to get the bucket name.
    match = _BUCKET_OBJECT_REGEX.match(source_archive_url)
    if not match:
        raise Exception(
            'Can not parse source URL {}.'.format(source_archive_url))
    bucket_name = match.group('bucket')
    object_name = match.group('object')

    bucket = client.bucket(bucket_name)
    blob = bucket.blob(object_name)
    try:
        blob.delete()
    except exceptions.NOT_FOUND:
        print('Cloud Function source archive URL {} does not exist, '
              'skipping delete.'.format(source_archive_url))

    try:
        # TODO(b/63595663): Remove only buckets that are created by cloud
        # ingest, this can be done by labeling buckets created by cloud ingest.
        bucket.delete()
    except exceptions.Conflict:
        print 'Bucket {} is not empty, skipping delete.'.format(bucket_name)


# TODO(b/63363798): Create unit tests for CloudFunctionsBuilder.
class CloudFunctionsBuilder(object):
    """Manipulates creation/deletion of cloud functions."""

    def __init__(self, location='us-central1'):
        credentials, self.project_id = googleauth.default()
        self.authed_session = AuthorizedSession(credentials)

        self.headers = {
            'Content-Type': 'application/json',
        }

        self.functions_path = 'projects/{}/locations/{}/functions'.format(
            self.project_id, location)
        self.functions_endpoint = (
            'https://cloudfunctions.googleapis.com/v1beta2')

        # Used to upload the cloud function source code to GCS.
        self.storage_client = storage.Client()

    # pylint: disable=too-many-arguments,too-many-locals
    # TODO(b/65407745): Reduce the number of arguments in
    # CloudFunctionsBuilder.create_function
    def create_function(self,
                        cloud_function_name,
                        src_dir,
                        pubsub_topic,
                        entry_point,
                        cloud_function_timeout,
                        staging_gcs_bucket=None,
                        staging_gcs_object=None,
                        timeout_seconds=180):
        """Creates a cloud function."""
        if not staging_gcs_object:
            staging_gcs_object = '%s_code.zip' % cloud_function_name
        src_zip_path = os.path.join(tempfile.gettempdir(), staging_gcs_object)
        _create_source_zip(src_dir, src_zip_path)

        if not staging_gcs_bucket:
            bucket = _create_function_bucket(self.storage_client,
                                             cloud_function_name)
        else:
            bucket = self.storage_client.get_bucket(staging_gcs_bucket)
        blob = bucket.blob(staging_gcs_object)
        blob.upload_from_filename(src_zip_path)

        # Upload the source code to GCS.
        functions_url = '{}/{}'.format(
            self.functions_endpoint, self.functions_path)
        payload = {
            'entryPoint': entry_point,
            'sourceArchiveUrl': 'gs://{}/{}'.format(bucket.name,
                                                    staging_gcs_object),
            'eventTrigger': {
                'resource': 'projects/{}/topics/{}'.format(self.project_id,
                                                           pubsub_topic),
                'eventType': 'providers/cloud.pubsub/eventTypes/topic.publish'
            },
            'name': '{}/{}'.format(self.functions_path, cloud_function_name),
            'timeout': cloud_function_timeout
        }
        res = self.authed_session.post(
            functions_url, headers=self.headers, json=payload)
        request_time = time.time()
        if res.status_code != httplib.OK:
            raise Exception('Unexpected error code when creating cloud '
                            'function {}, response text: {}.',
                            cloud_function_name, res.text)

        # Wait until the cloud function is ready.
        function_get_url = '{}/{}'.format(functions_url, cloud_function_name)
        while time.time() - request_time < timeout_seconds:
            print 'Waiting for cloud function to get deployed.'
            time.sleep(1)
            res = self.authed_session.get(function_get_url,
                                          headers=self.headers)
            if res.status_code != httplib.OK or 'status' not in res.json():
                # Something went wrong, skip checking the status of the cloud
                # function.
                continue
            if res.json()['status'] == 'READY':
                print 'Cloud function {} created in {} seconds.'.format(
                    cloud_function_name, time.time() - request_time)
                return
            elif res.json()['status'] == 'FAILED':
                raise Exception('Create cloud function {} failed.'.format(
                    cloud_function_name))

        raise Exception('Create cloud function {} timed out. Last query '
                        'response code: {}, response text: {}.'.format(
                            cloud_function_name, res.status_code, res.text))

    # pylint: enable=too-many-arguments,too-many-locals
    def delete_function(self, cloud_function_name, timeout_seconds=180):
        """Deletes a cloud function."""
        function_url = '{}/{}/{}'.format(
            self.functions_endpoint, self.functions_path, cloud_function_name)

        # Get the cloud function details before deleting it.
        res = self.authed_session.get(function_url, headers=self.headers)

        if res.status_code == httplib.NOT_FOUND:
            print 'Cloud Function {} does not exist, skipping delete.'.format(
                cloud_function_name)
            return
        elif res.status_code != httplib.OK:
            raise Exception(
                'Unexpected error code in getting cloud function {} details, '
                'response text: {}.', cloud_function_name, res.text)

        source_archive_url = res.json()['sourceArchiveUrl']

        res = self.authed_session.delete(function_url, headers=self.headers)
        request_time = time.time()

        if res.status_code != httplib.OK:
            raise Exception('Unexpected error code on deleteing cloud function '
                            '{}, response text: {}.',
                            cloud_function_name, res.text)

        # Polling until the cloud function get deleted.
        while (res.status_code != httplib.NOT_FOUND and
               time.time() - request_time < timeout_seconds):
            print 'Waiting for cloud function to get deleted.'
            time.sleep(1)
            res = self.authed_session.get(function_url, headers=self.headers)

        if res.status_code != httplib.NOT_FOUND:
            raise Exception('Delete cloud function {} timed out.'.format(
                cloud_function_name))

        _delete_function_bucket(self.storage_client, source_archive_url)

        print 'Cloud function {} deleted in {} seconds.'.format(
            cloud_function_name, time.time() - request_time)
