# Copyright 2019 Google Inc. All Rights Reserved.
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

"""Tests for autoupdate."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import os
import unittest
from absl import flags
import autoupdate
import mock

TEST_OBJECT_HAS_VERSION = 'TEST_OBJECT_HAS_VERSION'
TEST_OBJECT_MISSING_VERSION = 'TEST_OBJECT_MISSING_VERSION'
TEST_OBJECT_NOT_EXIST = 'TEST_OBJECT_NOT_EXIST'

FLAGS = flags.FLAGS


class MockPopen(object):

  def __init__(self, pid, return_code):
    self.pid = pid
    self.return_code = return_code

  def pid(self):
    return self.pid

  def poll(self):
    return self.return_code


class MockResponse(object):

  def __init__(self, status_code, json_data):
    self.status_code = status_code
    self.json_data = json_data

  def json(self):
    return self.json_data


def mock_requests_get(*args, **keywargs):
  del keywargs
  if args[0] == TEST_OBJECT_HAS_VERSION:
    return MockResponse(200, {'metadata': {'Version': TEST_OBJECT_HAS_VERSION}})
  elif args[0] == TEST_OBJECT_MISSING_VERSION:
    return MockResponse(200, {})
  elif args[0] == TEST_OBJECT_NOT_EXIST:
    return MockResponse(404, {})
  return MockResponse(404, None)


def create_agent_update_source_file(pid, text):
  filename = '/tmp/agent_source_%d.txt' % pid

  with open(filename, 'w+') as agent_source:
    agent_source.write(text)


def delete_agent_update_source_file(pid):
  filename = '/tmp/agent_source_%d.txt' % pid
  os.remove(filename)


class AgentReleaseVersionTest(unittest.TestCase):

  @mock.patch('requests.get', side_effect=mock_requests_get)
  def testAgentReleaseVersion_Successful(self, _):
    want = TEST_OBJECT_HAS_VERSION
    got = autoupdate.agent_release_version(TEST_OBJECT_HAS_VERSION)
    self.assertEqual(got, want)

  @mock.patch('requests.get', side_effect=mock_requests_get)
  def testAgentReleaseVersionMissing_Successful(self, _):
    got = autoupdate.agent_release_version(TEST_OBJECT_MISSING_VERSION)
    self.assertIsNone(got)

  @mock.patch('requests.get', side_effect=mock_requests_get)
  def testAgentReleaseVersionObjectMissing_Successful(self, _):
    got = autoupdate.agent_release_version(TEST_OBJECT_NOT_EXIST)
    self.assertIsNone(got)


class CheckProcessTest(unittest.TestCase):

  def testCheckProcessRunning_Successful(self):
    self.assertTrue(autoupdate.is_process_alive(MockPopen(1, None)))

  def testCheckProcessTerminated_Successful(self):
    self.assertFalse(autoupdate.is_process_alive(MockPopen(1, 0)))

  def testCheckProcess_Failed(self):
    self.assertFalse(autoupdate.is_process_alive(None))


class UpdateURLTest(unittest.TestCase):

  def setUp(self):
    super(UpdateURLTest, self).setUp()
    FLAGS(['autoupdate.py'], known_only=True)

  def testUpdateURL_Successful(self):
    mock_process = MockPopen(1, None)
    want = 'Test agent update source URL'
    create_agent_update_source_file(mock_process.pid, want)

    got = autoupdate.update_url(mock_process)
    self.assertEqual(got, want)

    delete_agent_update_source_file(mock_process.pid)

  def testReadAgentFileMissingProcess_Successful(self):
    mock_process = MockPopen(1, None)
    want = autoupdate.STABLE_AGENT_BINARY_ADDRESS

    got = autoupdate.update_url(mock_process)
    self.assertEqual(got, want)


if __name__ == '__main__':
  unittest.main()