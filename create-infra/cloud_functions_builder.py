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
import tempfile
import time
import zipfile

import google.auth as googleauth
from google.auth.transport.requests import AuthorizedSession
from google.cloud import storage


def CreateSourceZip(src_dir, zip_file_path):
  """Create source code zip file from src_dir."""
  with zipfile.ZipFile(zip_file_path, 'w') as zip_file:
    for root, _, files in os.walk(src_dir):
      for f in files:
        file_path = os.path.join(root, f)
        zip_file.write(file_path,
                       arcname=os.path.relpath(file_path, src_dir))


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
    self.functions_endpoint = 'https://cloudfunctions.googleapis.com/v1beta2'

    # Used to upload the cloud function source code to GCS.
    self.storage_client = storage.Client()

  def CreateFunction(self, cloud_function_name, src_dir,
                     staging_gcs_bucket, staging_gcs_object,
                     pubsub_topic,
                     entry_point):
    """Creates a cloud function."""
    src_zip_path = os.path.join(tempfile.gettempdir(), staging_gcs_object)
    CreateSourceZip(src_dir, src_zip_path)

    bucket = self.storage_client.get_bucket(staging_gcs_bucket)
    blob = bucket.blob(staging_gcs_object)
    blob.upload_from_filename(src_zip_path)

    # Upload the source code to GCS.
    functions_url = '{}/{}'.format(
        self.functions_endpoint, self.functions_path)
    payload = {
        'entryPoint': entry_point,
        'sourceArchiveUrl': 'gs://{}/{}'.format(staging_gcs_bucket,
                                                staging_gcs_object),
        'eventTrigger': {
            'resource': 'projects/{}/topics/{}'.format(self.project_id,
                                                       pubsub_topic),
            'eventType': 'providers/cloud.pubsub/eventTypes/topic.publish'
        },
        'name': '{}/{}'.format(self.functions_path, cloud_function_name)
    }
    r = self.authed_session.post(
        functions_url, headers=self.headers, json=payload)
    if r.status_code != httplib.OK:
      raise Exception('Unexpected error code when creating cloud function {}, '
                      'response text: {}.', cloud_function_name, r.text)

  def DeleteFunction(self, cloud_function_name, timeout_seconds=180):
    """Deletes a cloud function."""
    function_url = '{}/{}/{}'.format(
        self.functions_endpoint, self.functions_path, cloud_function_name)
    r = self.authed_session.delete(function_url, headers=self.headers)
    request_time = time.time()

    if r.status_code == httplib.NOT_FOUND:
      print 'Cloud Function {} does not exist, skipping delete.'.format(
          cloud_function_name)
    elif r.status_code != httplib.OK:
      raise Exception('Unexpected error code on deleteing cloud function {}, '
                      'response text: {}.', cloud_function_name, r.text)

    # Polling until the cloud function get deleted.
    while r.status_code != httplib.NOT_FOUND and (time.time() - request_time <
                                                  timeout_seconds):
      print 'Waiting for cloud function to get deleted.'
      time.sleep(1)
      r = self.authed_session.get(function_url, headers=self.headers)

    if r.status_code != httplib.NOT_FOUND:
      raise Exception(
          'Delete cloud function {} timed out.'.format(cloud_function_name))

    print 'Cloud function {} deleted in {} seconds.'.format(
        cloud_function_name, time.time() - request_time)
