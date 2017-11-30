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
"""Unit tests for spannerwrapper.py.

Tests that data is processed and returned in the proper format. Also
tests that creation methods return the appropriate value according to
the presence or lack of exceptions. The Cloud Spanner client library is
mocked, so these tests do not cover connecting to Cloud Spanner.
"""
import json
import logging
import unittest

from google.cloud.exceptions import Conflict
from google.gax import GaxError  # pylint: disable=relative-import
from grpc import StatusCode
from mock import MagicMock
from mock import patch

from proto import tasks_pb2
from spannerwrapper import SpannerWrapper
# Disable pylint since pylint bug makes pylint think google.gax
# is a relative import. Fix has been merged and will be included in
# next version of pylint (current version 1.7.2).
from test.testutils import get_rpc_error_with_status_code

JOB_CONFIG_ID_1 = u'test-config1'
JOB_CONFIG_ID_2 = u'test-config2'
JOB_SPEC_1 = {u'srcDir': u'usr/home/'}
JOB_SPEC_2 = {u'srcDir': u'usr/home2/'}
JOB_SPEC_STR_1 = '{"srcDir": "usr/home/"}'
JOB_SPEC_STR_2 = '{"srcDir": "usr/home2/"}'

class TestSpannerWrapper(unittest.TestCase):
    """Unit tests for spannerwrapper.py with the Cloud Spanner client mocked."""
    # pylint: disable=too-many-public-methods

    time_mock = MagicMock()
    time_mock.return_value = 12345

    @patch('spannerwrapper.spanner')
    # pylint: disable=arguments-differ
    def setUp(self, spanner_mock):
    # pylint: enable=arguments-differ
        logging.disable(logging.CRITICAL) # So tests don't print to console
        database = MagicMock()
        snapshot_obj = MagicMock()
        self.snapshot = MagicMock()

        self.spanner_instance = MagicMock()
        self.spanner_instance.database.return_value = database
        database.snapshot.return_value = snapshot_obj
        snapshot_obj.__enter__.return_value = self.snapshot

        self.spanner_client = MagicMock()
        self.spanner_client.instance.return_value = self.spanner_instance

        spanner_mock.Client.return_value = self.spanner_client

        self.pool = MagicMock()
        spanner_mock.BurstyPool.return_value = self.pool

        self.spanner_wrapper = SpannerWrapper('', '', '', '')

    def test_get_job_configs(self):
        """Asserts that two job configs are successfully returned."""
        result = MagicMock()
        result.__iter__.return_value = [[JOB_CONFIG_ID_1, JOB_SPEC_STR_1],
                                        [JOB_CONFIG_ID_2, JOB_SPEC_STR_2]]
        result.fields = self.get_fields_list(
            SpannerWrapper.JOB_CONFIGS_COLUMNS)
        self.snapshot.execute_sql.return_value = result

        actual = self.spanner_wrapper.get_job_configs()
        expected = [{u'JobConfigId': JOB_CONFIG_ID_1,
                     u'JobSpec': JOB_SPEC_1},
                    {u'JobConfigId': JOB_CONFIG_ID_2,
                     u'JobSpec': JOB_SPEC_2}]
        self.assertEqual(actual, expected)

    def test_get_configs_nonexistent(self):
        """Asserts that an empty list is returned when there are no configs."""
        result = MagicMock()
        result.__iter__.return_value = []
        self.snapshot.execute_sql.return_value = result

        actual = self.spanner_wrapper.get_job_configs()
        self.assertEqual(actual, [])

    def test_get_job_configs_table(self):
        """Asserts that the get_job_configs query uses the JobConfigs table."""
        self.spanner_wrapper.get_job_configs()
        self.snapshot.execute_sql.assert_called()
        query = self.snapshot.execute_sql.call_args[0][0]
        self.assertIn(SpannerWrapper.JOB_CONFIGS_TABLE, query)

    def test_get_job_config(self):
        """Asserts that a single job config is successfully returned."""
        config_id = 'test-config'
        job_spec = '{\'srcDir\': \'usr/home/\'}'

        result = MagicMock()
        result.__iter__.return_value = [[config_id, job_spec]]
        result.fields = self.get_fields_list(
            SpannerWrapper.JOB_CONFIGS_COLUMNS)
        self.snapshot.execute_sql.return_value = result

        actual = self.spanner_wrapper.get_job_config(config_id)
        expected = self.get_job_config(config_id, job_spec)

        self.assertEqual(actual, expected)

    def test_get_job_config_nonexistent(self):
        """Asserts that None is returned when there is no matching config."""
        config_id = 'test-config'

        result = MagicMock()
        result.__iter__.return_value = []
        self.snapshot.execute_sql.return_value = result

        actual = self.spanner_wrapper.get_job_config(config_id)
        self.assertIsNone(actual)

    def test_get_job_config_config_id(self):
        """Asserts that the proper JobConfigId is passed to the query."""
        config_id = 'test-config'
        self.spanner_wrapper.get_job_config(config_id)
        self.snapshot.execute_sql.assert_called()
        self.check_query_param(
            "config_id",
            config_id,
            self.snapshot.execute_sql.call_args
        )

    def test_create_job_config_table(self):
        """Asserts that create config uses the correct table and returns true.

        Asserts that create_job_config inserts into the JobConfigs table and
        returns true when no exception is raised."""
        transaction = self.set_up_transaction()
        self.spanner_wrapper.create_job_config('', '')
        transaction.insert.assert_called()
        table = transaction.insert.call_args[0][0]
        self.assertEqual(table, SpannerWrapper.JOB_CONFIGS_TABLE)

    def test_create_config_params(self):
        """Asserts that the correct values are passed to insert.

        Asserts that in create_job_config for each column the expected
        value is passed.
        """
        config_id = 'config-id'
        spec = 'spec'
        transaction = self.set_up_transaction()

        self.spanner_wrapper.create_job_config(config_id, spec)
        transaction.insert.assert_called()

        keyword_args = transaction.insert.call_args[1]
        columns = keyword_args['columns']
        # Since values is a list of objects being inserted, grab only the
        # first object
        values = keyword_args['values'][0]

        for i, column in enumerate(columns):
            if column == SpannerWrapper.JOB_CONFIG_ID:
                self.assertEqual(values[i], config_id)
            elif column == SpannerWrapper.JOB_SPEC:
                self.assertEqual(values[i], spec)
            else:
                self.fail("Tried to insert a value into a column that " +
                          "doesn't exist in %s. Column: %s" % (
                              SpannerWrapper.JOB_CONFIGS_TABLE, column))

        self.assertEqual(len(columns),
                         len(SpannerWrapper.JOB_CONFIGS_COLUMNS))
        self.assertEqual(len(values),
                         len(columns))


    def test_create_job_config_failure(self):
        """Asserts that create_job_config handles a raised GaxError.

        Tests that a GaxError raised by an insertion call to Cloud Spanner is
        handled correctly, resulting in create_job_config returning false.
        A GaxError is raised by the Cloud Spanner client in cases such as
        a duplicate id.
        """
        transaction = self.set_up_transaction()
        transaction.insert.side_effect = GaxError(
            "msg",
            get_rpc_error_with_status_code(StatusCode.ALREADY_EXISTS))

        self.assertRaises(
            Conflict,
            self.spanner_wrapper.create_job_config, 'config-id', 'spec')

    def test_create_job_run_table(self):
        """Asserts that create_job_run uses the correct table and returns true.

        Asserts that create_job_run inserts into the JobRuns table and
        returns true when no exception is raised."""
        transaction = self.set_up_transaction()
        self.spanner_wrapper.create_job_run('', '')
        transaction.insert.assert_called()
        table = transaction.insert.call_args[0][0]
        self.assertEqual(table, SpannerWrapper.JOB_RUNS_TABLE)

    @patch('time.time', time_mock)
    def test_create_run_params(self):
        """Asserts that the correct values are passed to insert.

        Asserts that in create_job_run for each column the expected
        value is passed. time.time() is mocked to always return the same
        value.
        """
        # pylint: disable=protected-access
        config_id = 'config-id'
        run_id = 'run-id'
        start_time = SpannerWrapper._get_unix_nano()
        counters = {
            'totalTasks': 0,
            'tasksCompleted': 0,
            'tasksFailed': 0,
            'tasksQueued': 0,
            'tasksUnqueued': 0,

            # List task stats.
            'totalTasksList': 0,
            'tasksCompletedList': 0,
            'tasksFailedList': 0,
            'tasksQueuedList': 0,
            'tasksUnqueuedList': 0,

            # Copy task stats.
            'totalTasksCopy': 0,
            'tasksCompletedCopy': 0,
            'tasksFailedCopy': 0,
            'tasksQueuedCopy': 0,
            'tasksUnqueuedCopy': 0
        }
        transaction = self.set_up_transaction()

        self.spanner_wrapper.create_job_run(config_id, run_id)
        transaction.insert.assert_called()
        keyword_args = transaction.insert.call_args[1]
        columns = keyword_args['columns']
        # Since values is a list of objects being inserted, grab only the
        # first object
        values = keyword_args['values'][0]

        for i, column in enumerate(columns):
            if column == SpannerWrapper.JOB_CONFIG_ID:
                self.assertEqual(values[i], config_id)
            elif column == SpannerWrapper.JOB_RUN_ID:
                self.assertEqual(values[i], run_id)
            elif column == SpannerWrapper.STATUS:
                # TODO(b/64227413): Replace 1 with enum or constant
                self.assertEqual(values[i], 1)
            elif column == SpannerWrapper.JOB_CREATION_TIME:
                # Test that the time inserted was the correct time
                self.assertEqual(values[i], start_time)
            elif column == SpannerWrapper.COUNTERS:
                self.assertEqual(json.loads(values[i]), counters)
            else:
                self.fail("Tried to insert a value into a column that " +
                          "doesn't exist in %s. Column: %s" % (
                              SpannerWrapper.JOB_RUNS_TABLE, column))

        self.assertEqual(len(columns),
                         len(SpannerWrapper.JOB_RUNS_COLUMNS))
        self.assertEqual(len(values),
                        len(columns))

    def test_create_run_failure(self):
        """Asserts that create_job_run handles a raised GaxError.

        Tests that a GaxError raised by an insertion call to Cloud Spanner is
        handled correctly, resulting in create_job_run returning false.
        A GaxError is raised by the Cloud Spanner client in cases such as
        a duplicate id.
        """
        transaction = self.set_up_transaction()
        transaction.insert.side_effect = GaxError(
            "msg",
            get_rpc_error_with_status_code(StatusCode.ALREADY_EXISTS))

        self.assertRaises(
            Conflict,
            self.spanner_wrapper.create_job_run, 'config-id', 'run_id')

    def test_get_tasks_of_status(self):
        """Asserts that the proper task status is used in the query"""
        task_status = tasks_pb2.TaskStatus.QUEUED
        self.spanner_wrapper.get_tasks_of_status(
            'fake_config_id', 'fake_run_id', 25, task_status)
        self.snapshot.execute_sql.assert_called()
        self.check_query_param(
            'task_status',
            task_status,
            self.snapshot.execute_sql.call_args)

    def test_last_modified_get_status(self):
        """get_tasks_of_status should populate the correct last_modified_before
           parameter in the query.
        """
        fake_last_modified = 1
        self.spanner_wrapper.get_tasks_of_status(
            'fake_config_id', 'fake_run_id', 25, tasks_pb2.TaskStatus.QUEUED,
            last_modified_before=fake_last_modified)
        self.snapshot.execute_sql.assert_called()
        self.check_query_param(
            'last_modified_before',
            fake_last_modified,
            self.snapshot.execute_sql.call_args)

    def test_get_tasks_of_failure_type(self):
        """Asserts that the proper failure type is used in the query"""
        task_failure_type = tasks_pb2.TaskFailureType.UNKNOWN
        # Get 25 tasks as it is the default number of tasks.
        self.spanner_wrapper.get_tasks_of_failure_type(
            'fake_config_id', 'fake_run_id', 25, task_failure_type)
        self.snapshot.execute_sql.assert_called()
        self.check_query_param(
            'failure_type',
            task_failure_type,
            self.snapshot.execute_sql.call_args)

    def test_last_modified_failure_type(self):
        """Asserts that the proper last modified is used in the query"""
        task_failure_type = tasks_pb2.TaskFailureType.UNKNOWN
        fake_last_modified = 1
        # Get 25 tasks as it is the default number of tasks.
        self.spanner_wrapper.get_tasks_of_failure_type(
            'fake_config_id', 'fake_run_id', 25, task_failure_type,
            fake_last_modified)
        self.snapshot.execute_sql.assert_called()
        self.check_query_param(
            'last_modified_before',
            fake_last_modified,
            self.snapshot.execute_sql.call_args)

    def set_up_transaction(self):
        """Sets up all needed mocks and returns the transaction mock.

        Creates all mocks needed for code in the following format:
          with self.session_pool.session() as session:
            with session.transaction() as transaction:

        Returns:
          The transaction mock so return values and side effects can be added.
        """
        transaction = MagicMock()

        transaction_context = MagicMock()
        transaction_context.__enter__.return_value = transaction

        session = MagicMock()
        session.transaction.return_value = transaction_context

        session_context = MagicMock()
        session_context.__enter__.return_value = session

        self.pool.session.return_value = session_context
        return transaction

    def check_query_param(self, param_name, expected_value, call_args):
        """Asserts that the query contains the given param with the right value.

        Asserts that the query contains a variable named param_name and that
        the variable param_name is passed the query with value expected_value.

        Args:
            param_name: The name of the variable in the query
            expected_value: The expected value of the param_name variable
            call_args: The call_args of the mocked execute_sql call
        """
        query = call_args[0][0]
        self.assertIn(param_name, query)

        query_params = call_args[0][1]
        self.assertIn(param_name, query_params)
        self.assertEqual(query_params[param_name], expected_value)

    @staticmethod
    def get_job_config(config_id, job_spec):
        """Returns a config in dictionary format containing the given values.

        Args:
          config_id: The job config id
          job_spec: The job spec in a JSON string

        Returns:
          A job config in dictionary format containing the given values.
        """
        return {
            SpannerWrapper.JOB_CONFIG_ID: config_id,
            SpannerWrapper.JOB_SPEC: job_spec
        }

    @staticmethod
    def get_fields_list(fields):
        """Returns fields in the format returned by the Cloud Spanner client.

        Returns a list of objects with the name property populated with the
        given fields.

        Args:
          fields: A list of strings representing the names of the fields.

        Returns:
          A list of fields like that returned by the Cloud Spanner client.
        """
        mocks = []
        for field in fields:
            field_mock = MagicMock()
            field_mock.name = field
            mocks.append(field_mock)
        return mocks


if __name__ == '__main__':
    unittest.main()
