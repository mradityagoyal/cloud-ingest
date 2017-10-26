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
"""Unit tests for main.py

Includes unit tests for the flask routes on main.py.
"""
import unittest
import main
from mock import patch
from mock import MagicMock
import json
from google.cloud.exceptions import Conflict
from google.cloud.exceptions import NotFound
from google.cloud.exceptions import Forbidden
from google.cloud.exceptions import PreconditionFailed
from google.cloud.exceptions import Unauthorized
from google.cloud.exceptions import BadRequest
from test.testutils import get_task
from proto import tasks_pb2

FAKE_TASK = get_task("fake_config_id",
                     "fake_run_id",
                     "fake_task_id",
                     task_creation_time=0,
                     last_mod_time=1,
                     status=tasks_pb2.TaskStatus.QUEUED,
                     task_spec="fake_task_spec",
                     task_type=tasks_pb2.TaskType.LIST)

class TestMain(unittest.TestCase):
    """Tests for main.py

    Includes tests for the routes in main.py.
    """
    # pylint: disable=too-many-public-methods

    def setUp(self):
        self.app = main.APP.test_client()
        self.app.testing = True

    @patch.object(main, '_get_credentials')
    def test_error_includes_traceback(self, _get_credentials_mock):
        """Tests that the common errors in main.py routes include a traceback"""
        def raise_exception_with_message(exception_class, message):
            """Raises the input exception class with input message."""
            raise exception_class(message)
        exception_list = [RuntimeError, BadRequest, Conflict, NotFound,
            Forbidden, PreconditionFailed, Unauthorized]
        expected_response_codes = [500, 400, 409, 404, 403, 412, 401]
        for i in range(0, len(exception_list)):
            def side_effect_function():
                """Side effect function for the mock used below"""
                raise_exception_with_message(exception_list[i], 'fake message')
            _get_credentials_mock.side_effect = side_effect_function
            response = self.app.get('/projects/fakeprojectid/jobconfigs')
            response_json = json.loads(response.data)
            self.assertEqual(response.status_code, expected_response_codes[i])
            self.assertTrue('fake message' in response_json['message'])
            self.assertTrue('Traceback' in response_json['traceback'])
            self.assertTrue('in raise_exception_with_message' in
              response_json['traceback'])
            self.assertTrue(exception_list[i].__name__ in
              response_json['traceback'])

    @patch.object(main, '_get_credentials')
    @patch.object(main, 'SpannerWrapper')
    def test_get_failure_type(self, spanner_wrapper_mock,
        dummy_get_credentials):
        """ Tests that getting tasks with failure type passes the correct
            parameters to spannerwrapper.
        """
        spanner_wrapper_mock_inst = MagicMock()
        spanner_wrapper_mock.return_value = spanner_wrapper_mock_inst
        spanner_wrapper_mock_inst.get_tasks_of_failure_type.return_value = \
            FAKE_TASK
        response = self.app.get(
            '/projects/fakeProjectId/tasks/fakeConfigId/fakeRunId/failuretype/'
            + str(tasks_pb2.TaskFailureType.UNKNOWN))
        response_json = json.loads(response.data)
        spanner_wrapper_mock_inst.get_tasks_of_failure_type.assert_called_with(
            'fakeConfigId', 'fakeRunId', main.DEFAULT_PAGE_SIZE,
            tasks_pb2.TaskFailureType.UNKNOWN)
        self.assertEqual(response_json, FAKE_TASK)

    @patch.object(main, '_get_credentials')
    @patch.object(main, 'SpannerWrapper')
    def test_get_tasks_of_status(self, spanner_wrapper_mock,
        dummy_get_credentials):
        """ Tests that getting tasks with status passes correct parameters to
            spannerwrapper.
        """
        spanner_wrapper_mock_inst = MagicMock()
        spanner_wrapper_mock.return_value = spanner_wrapper_mock_inst
        spanner_wrapper_mock_inst.get_tasks_of_status.return_value = FAKE_TASK
        response = self.app.get(
            '/projects/fakeProjectId/tasks/fakeConfigId/fakeRunId/status/'
            + str(tasks_pb2.TaskStatus.QUEUED))
        response_json = json.loads(response.data)
        spanner_wrapper_mock_inst.get_tasks_of_status.assert_called_with(
            'fakeConfigId', 'fakeRunId', main.DEFAULT_PAGE_SIZE,
            tasks_pb2.TaskStatus.QUEUED)
        self.assertEqual(response_json, FAKE_TASK)

    @patch.object(main, '_get_credentials')
    @patch.object(main, 'logging')
    def test_error_logs(self, _logging_mock, _get_credentials_mock):
        """Tests that the common errors are logged."""
        def raise_exception_with_message(exception_class, message):
            """Raises the input exception class with input message."""
            raise exception_class(message)
        exception_list = [RuntimeError, BadRequest, Conflict, NotFound,
            Forbidden, PreconditionFailed, Unauthorized]
        for i in range(0, len(exception_list)):
            def side_effect_function():
                """Side effect function for the mock used below."""
                raise_exception_with_message(exception_list[i], 'fake message')
            _get_credentials_mock.side_effect = side_effect_function
            self.app.post(('/projects/fakeprojectid/jobconfigs'
                           '?fakeparam=fakevalue'),
                           data=json.dumps(dict(
                           fake_field1='fake_content1',
                           fake_field2='fake_content2')),
                           content_type='application/json')
            last_call = _logging_mock.error.call_args_list[-1][0]
            logged_string = last_call[0] % last_call[1:]
            self.assertTrue('fake message' in logged_string)
            self.assertTrue('/projects/fakeprojectid/jobconfigs'
                in logged_string)
            self.assertTrue('"fakeparam": "fakevalue"' in logged_string)
            self.assertTrue('"fake_field1": "fake_content1"' in logged_string)
            self.assertTrue(exception_list[i].__name__ in logged_string)
            self.assertTrue('Traceback')

if __name__ == '__main__':
    unittest.main()
