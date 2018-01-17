# pylint: disable=too-many-lines
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
import httplib
import json
import unittest

from copy import deepcopy
from google.cloud import spanner
from google.cloud.exceptions import BadRequest
from google.cloud.exceptions import Conflict
from google.cloud.exceptions import Forbidden
from google.cloud.exceptions import NotFound
from google.cloud.exceptions import PreconditionFailed
from google.cloud.exceptions import Unauthorized
from google.cloud.spanner_v1.database import Database
from google.cloud.spanner_v1.database import SnapshotCheckout
from google.cloud.spanner_v1.instance import Instance
from google.cloud.spanner_v1.pool import SessionCheckout
from google.cloud.spanner_v1.proto import type_pb2
from google.cloud.spanner_v1.session import Session
from google.cloud.spanner_v1.snapshot import Snapshot
from google.cloud.spanner_v1.streamed import StreamedResultSet
from google.cloud.spanner_v1.transaction import Transaction
from mock import ANY
from mock import MagicMock
from mock import call
from mock import patch
from googleapiclient import discovery
from random import shuffle

import main
from create_infra import constants
from proto import tasks_pb2
from spannerwrapper import SpannerWrapper
from proto.tasks_pb2 import TaskFailureType
from proto.tasks_pb2 import TaskStatus

FAKE_JOB_SPEC = {
    'onPremSrcDirectory' : 'fakeFileSystemDir',
    'gcsBucket' : 'fakeGcsBucket'
}

FAKE_JOB_SPEC2 = {
    'onPremSrcDirectory' : 'fakeFileSystemDir2',
    'gcsBucket' : 'fakeGcsBucket2'
}

FAKE_JOB_SPEC_JSON = json.dumps(FAKE_JOB_SPEC)
FAKE_JOB_SPEC_JSON2 = json.dumps(FAKE_JOB_SPEC2)

FAKE_JOB_CONFIG_REQUEST = json.dumps({
    'jobConfigId' : 'fakeConfigId',
    'gcsBucket' : 'fakeGcsBucket',
    'fileSystemDirectory' : 'fakeFileSystemDir'})

FAKE_JOB_CONFIG_RESPONSE = {
    'JobConfigId' : 'fakeConfigId',
    'JobSpec' : FAKE_JOB_SPEC_JSON
}

FAKE_JOB_CONFIG_RESPONSE2 = {
    'JobConfigId' : 'fakeConfigId2',
    'JobSpec' : FAKE_JOB_SPEC_JSON2
}

# Fake Counters object containing what is expected on the job counters field.
FAKE_COUNTERS = {
    'tasksCompleted':0,
    'tasksCompletedCopy':0,
    'tasksCompletedList':0,
    'tasksCompletedLoad':0,
    'tasksFailed':0,
    'tasksFailedCopy':0,
    'tasksFailedList':0,
    'tasksFailedLoad':0,
    'tasksQueued':1,
    'tasksQueuedList':1,
    'totalTasks':1,
    'totalTasksCopy':0,
    'totalTasksList':1,
    'totalTasksLoad':0
}
FAKE_COUNTERS_JSON = json.dumps(FAKE_COUNTERS)

# Fake response returning information from a join in the JobConfigs and JobRuns
# table.
FAKE_JOB_RUN_JOB_CONFIG_RESPONSE = {
    'JobConfigId' : 'fakeConfigId',
    'JobRunId' : 'fakeJobRunId',
    'JobSpec' : FAKE_JOB_SPEC_JSON,
    'Counters' : FAKE_COUNTERS_JSON
}

FAKE_TIME1 = 1

# A dictionary representing a task spec stored in the backend.
FAKE_TASK_SPEC1 = {
    'src_directory': '/fake/source/directory/1',
    'dst_list_result_bucket': 'fake-destination-bucket-1',
    'dst_list_result_object': 'list-task-output-fake',
    'expected_generation_num': 0,
}

FAKE_TASK_SPEC1_JSON = json.dumps(FAKE_TASK_SPEC1)

# A dictionary representing a task that the backend returns.
FAKE_TASK1 = {
    SpannerWrapper.JOB_CONFIG_ID: 'fake config 1',
    SpannerWrapper.JOB_RUN_ID: 'jobRunId1',
    SpannerWrapper.TASK_ID: 'faketaskid',
    SpannerWrapper.TASK_CREATION_TIME: FAKE_TIME1,
    SpannerWrapper.LAST_MODIFICATION_TIME: FAKE_TIME1,
    SpannerWrapper.STATUS: tasks_pb2.TaskStatus.UNQUEUED,
    SpannerWrapper.TASK_SPEC: FAKE_TASK_SPEC1_JSON,
    SpannerWrapper.TASK_TYPE: tasks_pb2.TaskType.LIST
}

# A dictionary representing the expected flask output for the fake task 1.
EXPECTED_FAKE_TASK1 = {
    SpannerWrapper.JOB_CONFIG_ID: 'fake config 1',
    SpannerWrapper.JOB_RUN_ID: 'jobRunId1',
    SpannerWrapper.TASK_ID: 'faketaskid',
    SpannerWrapper.TASK_CREATION_TIME: str(FAKE_TIME1),
    SpannerWrapper.LAST_MODIFICATION_TIME: str(FAKE_TIME1),
    SpannerWrapper.STATUS: tasks_pb2.TaskStatus.UNQUEUED,
    SpannerWrapper.TASK_SPEC: FAKE_TASK_SPEC1,
    SpannerWrapper.TASK_TYPE: tasks_pb2.TaskType.LIST
}

FAKE_TIME2 = 2

# A dictionary representing a task spec stored in the backend.
FAKE_TASK_SPEC2 = {
    'src_directory': '/fake/source/directory/2',
    'dst_list_result_bucket': 'fake-destination-bucket-2',
    'dst_list_result_object': 'list-task-output-fake-2',
    'expected_generation_num': 0,
}

FAKE_TASK_SPEC2_JSON = json.dumps(FAKE_TASK_SPEC2)

# A dictionary representing a task stored in the backend.
FAKE_TASK2 = {
    SpannerWrapper.JOB_CONFIG_ID: 'fake config 2',
    SpannerWrapper.JOB_RUN_ID: 'jobRunId2',
    SpannerWrapper.TASK_ID: 'faketaskid2',
    SpannerWrapper.TASK_CREATION_TIME: FAKE_TIME2,
    SpannerWrapper.LAST_MODIFICATION_TIME: FAKE_TIME2,
    SpannerWrapper.STATUS: tasks_pb2.TaskStatus.QUEUED,
    SpannerWrapper.TASK_SPEC: FAKE_TASK_SPEC2_JSON,
    SpannerWrapper.TASK_TYPE: tasks_pb2.TaskType.LIST
}

# A dictionary representing the expected flask output for the fake task 1.
EXPECTED_FAKE_TASK2 = {
    SpannerWrapper.JOB_CONFIG_ID: 'fake config 2',
    SpannerWrapper.JOB_RUN_ID: 'jobRunId2',
    SpannerWrapper.TASK_ID: 'faketaskid2',
    SpannerWrapper.TASK_CREATION_TIME: str(FAKE_TIME2),
    SpannerWrapper.LAST_MODIFICATION_TIME: str(FAKE_TIME2),
    SpannerWrapper.STATUS: tasks_pb2.TaskStatus.QUEUED,
    SpannerWrapper.TASK_SPEC: FAKE_TASK_SPEC2,
    SpannerWrapper.TASK_TYPE: tasks_pb2.TaskType.LIST
}

FAKE_TASK_LIST = [FAKE_TASK1, FAKE_TASK2]

FAKE_INVALID_TIME = -4

_EMPTY_MOCK_STREAMED_RESULT = MagicMock(spec=StreamedResultSet)
_EMPTY_MOCK_STREAMED_RESULT.__iter__.return_value = []
_EMPTY_MOCK_STREAMED_RESULT.fields = []

# A testIamPermissions response from a cloudresourcemanager resource indicating
# that it has all permissions.
_ALL_PERMISSIONS_RESPONSE = {
    'permissions' : [
        'resourcemanager.projects.delete',
        'resourcemanager.projects.get',
        'resourcemanager.projects.update'
    ]
}

def _get_fields_list(names):
    """Gets a list of mock StrucType.Field from a list of names to populate
    these Fields with.
    """
    field_mocks = []
    for name in names:
        field_mock = MagicMock(spec=type_pb2.StructType.Field)
        field_mock.name = name
        field_mocks.append(field_mock)
    return field_mocks

def _get_mock_streamed_result(dictionary):
    """Gets a mock StreamedResultSet from an input dictionary.
    """
    mock_result = MagicMock(spec=StreamedResultSet)
    mock_result.__iter__.return_value = [
        dictionary.values()
    ]
    mock_result.one.return_value = dictionary.values()
    mock_result.fields = _get_fields_list(dictionary.keys())
    return mock_result

def _get_mock_streamed_result_list(dictionary_list):
    """Gets a mockStreamedResultSet from an input dictionary list.

    Args:
      dictionary_list: All the items in this list must have the same keys.
    """
    mock_result = MagicMock(spec=StreamedResultSet)
    mock_result.fields = _get_fields_list(dictionary_list[0].keys())
    result_list = []
    for dictionary in dictionary_list:
        result_list.append(dictionary.values())
    mock_result.__iter__.return_value = result_list
    return mock_result

class TestMain(unittest.TestCase):
    """Tests for main.py

    Includes tests for the routes in main.py.
    """
    # pylint: disable=too-many-public-methods
    # pylint: disable=too-many-instance-attributes
    def setUp(self):
        # pylint: disable=no-member
        self.app = main.APP.test_client()
        self.app.testing = True
        # Set up the spanner mocks
        self.mock_database = MagicMock(spec=Database)
        self.mock_instance = MagicMock(spec=Instance)
        self.mock_client = MagicMock(spec=spanner.Client)
        self.mock_transaction = MagicMock(spec=Transaction)
        self.mock_snapshot = MagicMock(spec=Snapshot)
        self.mock_bursty_pool = MagicMock(spec=spanner.BurstyPool)
        self.mock_session = MagicMock(spec=Session)
        self.mock_session_checkout = MagicMock(spec=SessionCheckout)
        self.snapshot_checkout_mock = MagicMock(spec=SnapshotCheckout)
        # Start the patchers used in all the tests.
        self.spanner_mock_patcher = patch('spannerwrapper.spanner')
        self.spanner_mock = self.spanner_mock_patcher.start()
        self.credentials_mock = MagicMock()
        (self.
         get_credentials_mock_patcher) = patch.object(main, '_get_credentials')
        self.get_credentials_mock = self.get_credentials_mock_patcher.start()
        self.get_credentials_mock.return_value = self.credentials_mock
        # Start discovery patcher
        self.discovery_patcher = patch.object(main, 'discovery',
            spec=discovery)
        self.discovery_mock = self.discovery_patcher.start()
        # Make a cloudresource manager discovery.Resource object
        self.resource_mock = MagicMock(spec=['projects'])
        self.discovery_mock.build.return_value = self.resource_mock
        self.resource_mock_projects = MagicMock()
        self.mock_projects_request = MagicMock(spec=['execute'])
        (self.resource_mock_projects.testIamPermissions.
         return_value) = self.mock_projects_request
        (self.resource_mock.projects.
         return_value) = self.resource_mock_projects
        # Return that the user has all permissions.
        (self.mock_projects_request.execute.
         return_value) = _ALL_PERMISSIONS_RESPONSE
        # Set up the client and pool
        self.spanner_mock.Client.return_value = self.mock_client
        self.spanner_mock.BurstyPool.return_value = self.mock_bursty_pool
        self.mock_client.instance.return_value = self.mock_instance
        # Set up the database
        self.mock_instance.database.return_value = self.mock_database
        self.snapshot_checkout_mock.__enter__.return_value = self.mock_snapshot
        self.mock_database.snapshot.return_value = self.snapshot_checkout_mock
        self.mock_session_checkout.__enter__.return_value = self.mock_session
        self.mock_bursty_pool.session.return_value = self.mock_session_checkout
        # Set up the transactions
        self.mock_transaction.return_value = self.mock_transaction
        self.mock_transaction.__enter__ = self.mock_transaction
        self.mock_session.transaction.return_value = self.mock_transaction
        def run_in_transaction(trans_function, *args):
            """A replacement for the run_in_transaction function that runs the
               function that is passed.
            """
            trans_function(self.mock_transaction, *args)
        self.mock_database.run_in_transaction = run_in_transaction

        # pylint: disable=protected-access
        # Re-initialize the spanner wrapper to get the mock in effect.
        main._SPANNER_WRAPPER = SpannerWrapper(
            main._CREDENTIALS, main._HOST_PROJECT,
            main.APP.config['SPANNER_INSTANCE'],
            main.APP.config['SPANNER_DATABASE'])
        # pylint: enable=protected-access

    @staticmethod
    @patch.object(main, 'PubSubBuilder')
    def test_create_pubsub_not_exists(builder_mock):
        """Test _create_pubsub_if_not_exists only creates the non existing
        Pub/Sub topics and subscriptions.
        """
        # pylint: disable=protected-access,no-member
        def pub_sub_exist_mock(topic, _):
            """Mock for Pub/Sub topic/subscription existance."""
            if topic == constants.LIST_TOPIC:
                return False
            return True

        bldr_object = MagicMock()
        builder_mock.return_value = bldr_object
        bldr_object.topic_and_subscriptions_exist.side_effect = (
            pub_sub_exist_mock)

        main._create_pubsub_if_not_exists("credentials", "project")
        builder_mock.assert_called_once_with(
            credentials="credentials", project_id="project")
        bldr_object.create_topic_and_subscriptions.assert_called_once_with(
            constants.LIST_TOPIC, [constants.LIST_SUBSCRIPTION],
            ack_deadline=30)
        # pylint: enable=protected-access,no-member

    def test_add_policy_binding(self):
        """Tests _add_policy_binding adds the binding and returns if it's added.
        """
        # pylint: disable=protected-access,no-member
        policy = {
            'bindings': [
                {
                  'members': ['member1'],
                  'role': 'role1'
                },
                {
                  'members': ['member1', 'member2'],
                  'role': 'role2'
                },
            ]
        }
        # Tests the binding already exists.
        self.assertFalse(main._add_policy_binding(policy, 'member2', 'role2'))

        # Tests the role exists but does not contain the member.
        self.assertTrue(main._add_policy_binding(policy, 'member3', 'role2'))
        self.assertFalse(main._add_policy_binding(policy, 'member3', 'role2'))

        # Tests when the role does not exists.
        self.assertTrue(main._add_policy_binding(policy, 'member1', 'role3'))
        self.assertFalse(main._add_policy_binding(policy, 'member1', 'role3'))

        expected_updated_policy = {
            'bindings': [
                {
                  'members': ['member1'],
                  'role': 'role1'
                },
                {
                  'members': ['member1', 'member2', 'member3'],
                  'role': 'role2'
                },
                {
                  'members': ['member1'],
                  'role': 'role3'
                },
            ]
        }
        self.assertEqual(json.dumps(policy, sort_keys=True),
            json.dumps(expected_updated_policy, sort_keys=True))
        # pylint: enable=protected-access,no-member

    def test_grant_sa_permissions(self):
        """Test _grant_service_account_permissions_if_needed grant the service
        account the permissions needed.
        """
        # pylint: disable=protected-access,no-member
        get_iam_policy_mock = MagicMock()
        set_iam_policy_mock = MagicMock()
        self.resource_mock_projects.getIamPolicy.return_value = (
            get_iam_policy_mock)
        self.resource_mock_projects.setIamPolicy.return_value = (
            set_iam_policy_mock)

        policy = {
            'bindings': [
                {
                  'members': ['member1'],
                  'role': 'role1'
                },
                {
                  'members': ['member1', 'member2'],
                  'role': 'role2'
                },
            ]
        }

        expected_updated_policy = deepcopy(policy)
        for role in main._SERVICE_ACCOUNT_ROLES:
            expected_updated_policy['bindings'].append({
                'members': [main._SERVICE_ACCOUNT_MEMBER],
                'role': role
                })
        get_iam_policy_mock.execute.return_value = policy
        main._grant_service_account_permissions_if_needed(
            self.credentials_mock, 'fakeprojectid')
        get_iam_policy_mock.execute.assert_called_once_with()
        self.resource_mock_projects.setIamPolicy.assert_called_once_with(
            resource='fakeprojectid', body={'policy': expected_updated_policy})
        set_iam_policy_mock.execute.assert_called_once_with()
        # pylint: enable=protected-access,no-member

    def test_not_grant_sa_permissions(self):
        """Test _grant_service_account_permissions_if_needed find that the
        service account has the right permissions and does not update the
        policy.
        """
        # pylint: disable=protected-access,no-member
        get_iam_policy_mock = MagicMock()
        self.resource_mock_projects.getIamPolicy.return_value = (
            get_iam_policy_mock)

        policy = {
            'bindings': [
                {
                  'members': ['member1'],
                  'role': 'role1'
                },
                {
                  'members': ['member1', 'member2'],
                  'role': 'role2'
                },
            ]
        }
        for role in main._SERVICE_ACCOUNT_ROLES:
            policy['bindings'].append({
                'members': [main._SERVICE_ACCOUNT_MEMBER],
                'role': role
                })
        shuffle(policy['bindings'])

        get_iam_policy_mock.execute.return_value = policy
        main._grant_service_account_permissions_if_needed(
            self.credentials_mock, 'fakeprojectid')
        get_iam_policy_mock.execute.assert_called_once_with()
        self.resource_mock_projects.setIamPolicy.assert_not_called()
        # pylint: enable=protected-access,no-member

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

    def test_delete_job_configs(self):
        """
        jobconfigs/delete should return a bad request if there are any tasks
        in progress for any of the configs.
        """
        mock_streamed_result = MagicMock()
        # Result says: 2 tasks in progress for fakeconfigid1 and 0 tasks in
        # progress of fakeconfigid2.
        mock_streamed_result.rows = [[2], [0]]
        # Execute sql function requests the number of tasks in progress.
        self.mock_transaction.execute_sql.return_value = mock_streamed_result

        response = self.app.post(('/projects/fakeprojectid/jobconfigs/delete'),
                           data=json.dumps(['fakeconfigid1', 'fakeconfigid2']),
                           content_type='application/json')
        response_json = json.loads(response.data)

        self.mock_transaction.delete.assert_called() # Deleted config from db.

        assert response.status_code == httplib.BAD_REQUEST
        assert 'fakeconfigid1' in response_json['message']
        assert 'fakeconfigid2' not in response_json['message']

    def test_delete_job_configs_success(self):
        """
        jobconfigs/delete should return response status OK if all the configs
        were deleted successfully.
        """
        mock_streamed_result = MagicMock()
        # Result says: 0 tasks in progress for fakeconfigid1 and 0 tasks in
        # progress of fakeconfigid2.
        mock_streamed_result.rows = [[0], [0]]
        # Execute sql function requests the number of tasks in progress.
        self.mock_transaction.execute_sql.return_value = mock_streamed_result

        response = self.app.post(('/projects/fakeprojectid/jobconfigs/delete'),
                           data=json.dumps(['fakeconfigid1', 'fakeconfigid2']),
                           content_type='application/json')
        response_json = json.loads(response.data)

        self.mock_transaction.delete.assert_called() # Deleted config from db.

        assert response.status_code == httplib.OK
        assert 'fakeconfigid1' in response_json
        assert 'fakeconfigid2' in response_json

    def test_no_deleted_configs_error(self):
        """
        jobconfigs/delete should not try to delete anything if there are tasks
        in progress for all the configs.
        """
        mock_streamed_result = MagicMock()
        # Result says: 2 tasks in progress for fakeconfigid1 and 2 tasks in
        # progress of fakeconfigid2.
        mock_streamed_result.rows = [[2], [2]]
        # Execute sql function requests the number of tasks in progress.
        self.mock_transaction.execute_sql.return_value = mock_streamed_result

        response = self.app.post(('/projects/fakeprojectid/jobconfigs/delete'),
                           data=json.dumps(['fakeconfigid1', 'fakeconfigid2']),
                           content_type='application/json')

        response_json = json.loads(response.data)
        # Should not delete anything from the database.
        self.mock_transaction.delete.assert_not_called()

        assert response.status_code == httplib.BAD_REQUEST
        assert 'fakeconfigid1' in response_json['message']
        assert 'fakeconfigid2' in response_json['message']

    def test_delete_job_configs_error(self):
        """
        jobconfigs/delete should return an error if the list of config ids
        is not correctly formatted.
        """
        response = self.app.post(('/projects/fakeprojectid/jobconfigs/delete'),
                           data=json.dumps(dict(
                               fakeFieldInvalidFormat='InvalidField'
                           )),
                           content_type='application/json')

        response_json = json.loads(response.data)
        assert response_json['error'] is not None

    def test_delete_job_configs_query(self):
        """
        jobconfigs/delete should make an sql query to select the count of tasks
        with job config equal to the input configs
        """
        self.app.post(('/projects/fakeprojectid/jobconfigs/delete'),
            data=json.dumps(['fakeconfigid1', 'fakeconfigid2',
            'fakeconfigid3']), content_type='application/json')

        sql_query = self.mock_transaction.execute_sql.call_args[0][0]
        params = self.mock_transaction.execute_sql.call_args[1]['params']
        param_types = (self
            .mock_transaction.execute_sql.call_args[1]['param_types'])

        # Assert that the query is counting from tasks table.
        assert 'SELECT COUNT' in sql_query
        assert 'FROM {0}'.format(SpannerWrapper.TASKS_TABLE) in sql_query

        # Assert that the params and param types appear in the query.
        assert ('{0} = @config_0'.format(SpannerWrapper.JOB_CONFIG_ID)
            in sql_query)
        assert ('{0} = @config_1'.format(SpannerWrapper.JOB_CONFIG_ID)
            in sql_query)
        assert ('{0} = @config_2'.format(SpannerWrapper.JOB_CONFIG_ID)
            in sql_query)

        assert (params['config_0'] == 'fakeconfigid1'
            and 'config_0' in param_types)
        assert (params['config_1'] == 'fakeconfigid2'
            and 'config_1' in param_types)
        assert (params['config_2'] == 'fakeconfigid3'
            and 'config_2' in param_types)

    def test_get_job_run(self):
        """
        jobrun/<config_id> should return the job run and job config info from
        the job run.
        """
        mock_result = _get_mock_streamed_result(
            FAKE_JOB_RUN_JOB_CONFIG_RESPONSE)
        self.mock_snapshot.execute_sql.return_value = mock_result
        response = self.app.get('/projects/fakeprojectid/jobrun/fakeconfigid')
        response_json = json.loads(response.data)
        sql_query = self.mock_snapshot.execute_sql.call_args[0][0]
        params = self.mock_snapshot.execute_sql.call_args[0][1]

        # Assert that the query reads from both the job runs and job configs
        # table.
        assert 'FROM {0} JOIN {1}'.format(SpannerWrapper.JOB_RUNS_TABLE,
            SpannerWrapper.JOB_CONFIGS_TABLE) in sql_query
        assert ('{0} = @config_id'.format(SpannerWrapper.JOB_CONFIG_ID)
            in sql_query)
        assert ('{0} = @run_id'.format(SpannerWrapper.JOB_RUN_ID)
            in sql_query)

        # Assert that the query passes the expected config parameters.
        assert params['config_id'] == 'fakeconfigid'
        assert params['run_id'] == 'jobrun'

        # Assert that the response json is the result from the query.
        assert (response_json[SpannerWrapper.JOB_CONFIG_ID] ==
            FAKE_JOB_RUN_JOB_CONFIG_RESPONSE[SpannerWrapper.JOB_CONFIG_ID])
        assert (response_json[SpannerWrapper.JOB_RUN_ID] ==
            FAKE_JOB_RUN_JOB_CONFIG_RESPONSE[SpannerWrapper.JOB_RUN_ID])
        assert (response_json[SpannerWrapper.COUNTERS] ==
            json.loads(
                FAKE_JOB_RUN_JOB_CONFIG_RESPONSE[SpannerWrapper.COUNTERS]))
        assert (response_json[SpannerWrapper.JOB_SPEC] ==
            json.loads(
                FAKE_JOB_RUN_JOB_CONFIG_RESPONSE[SpannerWrapper.JOB_SPEC]))

    def test_get_job_run_error1(self):
        """
        jobrun/<config_id> should return an error if the config id is not in the
        expected format.
        """
        response = self.app.get('/projects/fakeprojectid/jobrun/'
                                'fake*invalid.config')
        response_json = json.loads(response.data)

        assert response.status_code == httplib.BAD_REQUEST
        assert response_json['error'] is not None

    def test_get_job_run_error2(self):
        """
        jobrun/<config_id> should return an error if the project id is not in
        the expected format.
        """
        response = self.app.get('/projects/fake*invalid+projectid/jobrun/'
                                'fakeconfigid')
        response_json = json.loads(response.data)

        assert response.status_code == httplib.BAD_REQUEST
        assert response_json['error'] is not None

    def test_job_run_not_found(self):
        """
        jobrun/<config_id> should return a not found error if the job config
        was not found.
        """
        # Return no jobs when reading from the database.
        (self.mock_snapshot.execute_sql
         .return_value) = _get_mock_streamed_result({})

        response = self.app.get('/projects/fakeprojectid/jobrun/'
                                'fakeconfigid')
        response_json = json.loads(response.data)

        assert response.status_code == httplib.NOT_FOUND
        assert response_json['error'] is not None

    def test_get_job_configs(self):
        """
        <project_id>/jobconfigs GET should return a list of job configurations
        from the spanner database.
        """
        mock_result = _get_mock_streamed_result_list([FAKE_JOB_CONFIG_RESPONSE,
            FAKE_JOB_CONFIG_RESPONSE2])
        self.mock_snapshot.execute_sql.return_value = mock_result

        response = self.app.get('/projects/fakeprojectid/jobconfigs')
        response_json = json.loads(response.data)

        sql_query = self.mock_snapshot.execute_sql.call_args[0][0]

        # Assert it reads from job configs table and job runs table.
        assert SpannerWrapper.JOB_CONFIGS_TABLE in sql_query
        assert SpannerWrapper.JOB_RUNS_TABLE in sql_query

        # Assert that both configs are returned.
        assert (response_json[0][SpannerWrapper.JOB_CONFIG_ID] ==
            FAKE_JOB_CONFIG_RESPONSE[SpannerWrapper.JOB_CONFIG_ID])
        assert (response_json[0][SpannerWrapper.JOB_SPEC] ==
            json.loads(FAKE_JOB_CONFIG_RESPONSE[SpannerWrapper.JOB_SPEC]))

        assert (response_json[1][SpannerWrapper.JOB_CONFIG_ID] ==
            FAKE_JOB_CONFIG_RESPONSE2[SpannerWrapper.JOB_CONFIG_ID])
        assert (response_json[1][SpannerWrapper.JOB_SPEC] ==
            json.loads(FAKE_JOB_CONFIG_RESPONSE2[SpannerWrapper.JOB_SPEC]))

    @patch.object(main, '_create_pubsub_if_not_exists')
    @patch.object(main, '_grant_service_account_permissions_if_needed')
    def test_post_job_configs(self, grant_sa_perm_mock, create_pubsub_mock):
        """
        <project_id>/jobconfigs POST should create a new job configuration.
        """
        self.app.post(('/projects/fakeprojectid/jobconfigs'),
            data=json.dumps({'jobConfigId' : 'fakeConfigId1',
                             'gcsBucket' : 'fake-gcs-bucket-1',
                             'fileSystemDirectory': '/fake/on/prem/dir'}),
                             content_type='application/json')

        self.mock_transaction.insert_or_update.assert_called_once_with(
            SpannerWrapper.PROJECTS_TABLE,
            columns=SpannerWrapper.PROJECTS_COLUMNS,
            values=[(
                "fakeprojectid", constants.LIST_TOPIC, constants.COPY_TOPIC,
                constants.LIST_PROGRESS_SUBSCRIPTION,
                constants.COPY_PROGRESS_SUBSCRIPTION
            )])

        calls = [
            call(SpannerWrapper.JOB_CONFIGS_TABLE,
                columns=SpannerWrapper.JOB_CONFIGS_COLUMNS,
                values=ANY),
            call(
                SpannerWrapper.JOB_RUNS_TABLE,
                columns=SpannerWrapper.JOB_RUNS_COLUMNS,
                values=ANY),
            call(
                SpannerWrapper.TASKS_TABLE,
                columns=SpannerWrapper.TASKS_COLUMNS,
                values=ANY)
        ]
        self.assertEqual(len(self.mock_transaction.insert.mock_calls), 3)
        self.mock_transaction.insert.assert_has_calls(calls)

        # Assert that the pubsub is created successfully.
        create_pubsub_mock.assert_called_once_with(
            self.credentials_mock, "fakeprojectid")
        grant_sa_perm_mock.assert_called_once_with(
            self.credentials_mock, "fakeprojectid")

    def test_post_job_configs_error(self):
        """
        <project_id>/jobconfigs POST should throw an error if the project id
        does not match the pattern.
        """
        response = self.app.post(('/projects/fakeprojectid/jobconfigs'),
            data=json.dumps({'jobConfigId' : 'invalid*config.name',
                             'gcsBucket' : 'fake-gcs-bucket-1',
                             'fileSystemDirectory': '/fake/on/prem/dir'}),
                             content_type='application/json')

        response_json = json.loads(response.data)

        assert response.status_code == httplib.BAD_REQUEST
        assert response_json['error'] is not None

    def test_post_job_configs_error2(self):
        """
        <project_id>/jobconfigs POST should throw an error if the bucket id does
        not match the pattern.
        """
        response = self.app.post(('/projects/fakeprojectid/jobconfigs'),
            data=json.dumps({'jobConfigId' : 'fakeConfigId1',
                             'gcsBucket' : 'invalid*gcs.bucket',
                             'fileSystemDirectory': '/fake/on/prem/dir'}),
                             content_type='application/json')

        response_json = json.loads(response.data)

        assert response.status_code == httplib.BAD_REQUEST
        assert response_json['error'] is not None

    def test_post_job_configs_error3(self):
        """
        <project_id>/jobconfigs POST should throw an error if the file system
        directory is not in a valid format.
        """
        response = self.app.post(('/projects/fakeprojectid/jobconfigs'),
            data=json.dumps({'jobConfigId' : 'fakeConfigId1',
                             'gcsBucket' : 'fake-gcs-bucket-1',
                             'fileSystemDirectory': '*not-a-valid-directory'}),
                             content_type='application/json')

        response_json = json.loads(response.data)
        assert response.status_code == httplib.BAD_REQUEST
        assert response_json['error'] is not None

    def test_get_tasks_failure_type(self):
        """
        /projects/<project_id>/tasks/<config_id>/failuretype/<failure_type>
        should get the tasks of the input failure type.
        """
        mock_result = _get_mock_streamed_result_list(FAKE_TASK_LIST)
        self.mock_snapshot.execute_sql.return_value = mock_result
        response = self.app.get('/projects/fakeprojectid/tasks/fakeconfigid/'
            'failuretype/' + str(TaskFailureType.FILE_MODIFIED_FAILURE) +
            '?lastModifiedBefore=' + str(FAKE_TIME1))
        response_json = json.loads(response.data)
        sql_query = self.mock_snapshot.execute_sql.call_args[0][0]
        params = self.mock_snapshot.execute_sql.call_args[0][1]

        # Assert that it returns the tasks from the query.
        assert response_json[0] == EXPECTED_FAKE_TASK1
        assert response_json[1] == EXPECTED_FAKE_TASK2

        # Assert the query uses the expected query parameters
        assert ('FROM {0}'.format(SpannerWrapper.TASKS_TABLE)
            in sql_query)
        assert ('{0} = @config_id'.format(SpannerWrapper.JOB_CONFIG_ID) in
            sql_query)
        assert ('{0} = @failure_type'.format(SpannerWrapper.FAILURE_TYPE) in
            sql_query)
        assert ('{0} < @last_modified_before'.format(
            SpannerWrapper.LAST_MODIFICATION_TIME) in sql_query)

        # Assert that it contains the query parameter values.
        assert params['config_id'] == 'fakeconfigid'
        assert params['failure_type'] == TaskFailureType.FILE_MODIFIED_FAILURE
        assert params['last_modified_before'] == FAKE_TIME1

    def test_get_tasks_failure_error(self):
        """
        /projects/<project_id>/tasks/<config_id>/failuretype/<failure_type>
        should return an error if the failure type is invalid
        """
        response = self.app.get('/projects/fakeprojectid/tasks/fakeconfigid/'
            'failuretype/500')
        response_json = json.loads(response.data)

        assert response.status_code == httplib.BAD_REQUEST
        assert response_json['error'] is not None

    def test_get_tasks_failure_error2(self):
        """
        /projects/<project_id>/tasks/<config_id>/failuretype/<failure_type>
        should return an error if the last modification time is invalid
        """
        response = self.app.get('/projects/fakeprojectid/tasks/fakeconfigid/'
            'failuretype/' + str(TaskFailureType.MD5_MISMATCH_FAILURE) +
            '?lastModifiedBefore=' + str(FAKE_INVALID_TIME))
        response_json = json.loads(response.data)

        assert response.status_code == httplib.BAD_REQUEST
        assert response_json['error'] is not None

    def test_get_tasks_failure_error3(self):
        """
        /projects/<project_id>/tasks/<config_id>/failuretype/<failure_type>
        should return an error if the project id is invalid
        """
        response = self.app.get('/projects/fake*.*invalid/tasks/fakeconfigid/'
            'failuretype/' + str(TaskFailureType.MD5_MISMATCH_FAILURE) +
            '?lastModifiedBefore=' + str(FAKE_INVALID_TIME))
        response_json = json.loads(response.data)

        assert response.status_code == httplib.BAD_REQUEST
        assert response_json['error'] is not None

    def test_get_tasks_failure_error4(self):
        """
        /projects/<project_id>/tasks/<config_id>/failuretype/<failure_type>
        should return an error if the config id is invalid
        """
        response = self.app.get('/projects/fakeprojectid/tasks/fake*invalid./'
            'failuretype/' + str(TaskFailureType.MD5_MISMATCH_FAILURE) +
            '?lastModifiedBefore=' + str(FAKE_INVALID_TIME))
        response_json = json.loads(response.data)

        assert response.status_code == httplib.BAD_REQUEST
        assert response_json['error'] is not None


    def test_get_tasks_of_status(self):
        """
        /projects/<project_id>/tasks/<config_id>/status/<task_status>
        should get the tasks of the input failure status.
        """
        mock_result = _get_mock_streamed_result_list(FAKE_TASK_LIST)
        self.mock_snapshot.execute_sql.return_value = mock_result
        response = self.app.get('/projects/fakeprojectid/tasks/fakeconfigid/'
            'status/' + str(TaskStatus.UNQUEUED) +
            '?lastModifiedBefore=' + str(FAKE_TIME1))
        response_json = json.loads(response.data)
        sql_query = self.mock_snapshot.execute_sql.call_args[0][0]
        params = self.mock_snapshot.execute_sql.call_args[0][1]

        # Assert that it returns the tasks from the query.
        assert response_json[0] == EXPECTED_FAKE_TASK1
        assert response_json[1] == EXPECTED_FAKE_TASK2

        # Assert the query uses the expected query parameters
        assert ('FROM {0}'.format(SpannerWrapper.TASKS_TABLE)
            in sql_query)
        assert ('{0} = @config_id'.format(SpannerWrapper.JOB_CONFIG_ID) in
            sql_query)
        assert ('{0} = @task_status'.format(SpannerWrapper.STATUS) in
            sql_query)
        assert ('{0} < @last_modified_before'.format(
            SpannerWrapper.LAST_MODIFICATION_TIME) in sql_query)

        # Assert that it contains the query parameter values.
        assert params['config_id'] == 'fakeconfigid'
        assert params['task_status'] == TaskStatus.UNQUEUED
        assert params['last_modified_before'] == FAKE_TIME1

    def test_get_tasks_of_status_error(self):
        """
        /projects/<project_id>/tasks/<config_id>/status/<task_status>
        should return an error if the status is invalid
        """
        response = self.app.get('/projects/fakeprojectid/tasks/fakeconfigid/'
            'status/500')
        response_json = json.loads(response.data)

        assert response.status_code == httplib.BAD_REQUEST
        assert response_json['error'] is not None

    def test_get_tasks_of_status_error2(self):
        """
        /projects/<project_id>/tasks/<config_id>/status/<task_status>
        should return an error if the last modification time is invalid
        """
        response = self.app.get('/projects/fakeprojectid/tasks/fakeconfigid/'
            'status/' + str(TaskStatus.UNQUEUED) +
            '?lastModifiedBefore=' + str(FAKE_INVALID_TIME))
        response_json = json.loads(response.data)

        assert response.status_code == httplib.BAD_REQUEST
        assert response_json['error'] is not None

    def test_get_tasks_of_status_error3(self):
        """
        /projects/<project_id>/tasks/<config_id>/status/<task_status>
        should return an error if the project id is invalid
        """
        response = self.app.get('/projects/invalid$projectid/tasks/'
            'fakeconfigid/status/' + str(TaskStatus.UNQUEUED) +
            '?lastModifiedBefore=' + str(FAKE_INVALID_TIME))
        response_json = json.loads(response.data)

        assert response.status_code == httplib.BAD_REQUEST
        assert response_json['error'] is not None

    def test_get_tasks_of_status_error4(self):
        """
        /projects/<project_id>/tasks/<config_id>/status/<task_status>
        should return an error if the config id is invalid
        """
        response = self.app.get('/projects/invalid$projectid/tasks/'
            'invalid#configid/status/' + str(TaskStatus.UNQUEUED) +
            '?lastModifiedBefore=' + str(FAKE_INVALID_TIME))
        response_json = json.loads(response.data)

        assert response.status_code == httplib.BAD_REQUEST
        assert response_json['error'] is not None

    def test_doesnt_have_permissions(self):
        """
        Tests that the user receives an error when the Project.exists function
        throws an error.
        """
        # Return that there are no permissions for the project.
        self.mock_projects_request.execute.return_value = {}
        response = self.app.get('/projects/fakeprojectid/jobconfigs')
        assert response.status_code == httplib.FORBIDDEN

    def tearDown(self):
        # Stop patchers.
        self.spanner_mock_patcher.stop()
        self.get_credentials_mock_patcher.stop()
        self.discovery_patcher.stop()

if __name__ == '__main__':
    unittest.main()
