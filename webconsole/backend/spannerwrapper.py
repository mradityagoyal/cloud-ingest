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
"""SpannerWrapper handles all interactions with Cloud Spanner.

SpannerWrapper lists JobConfigs, JobRuns, and Tasks. It
also writes new JobConfigs and JobRuns. All data passed to and from the
client is in JSON format and stored in a dictionary.
"""
import logging
import time
#pylint: disable=no-name-in-module, import-error, relative-import
from google.cloud import spanner
from google.cloud.proto.spanner.v1 import type_pb2
from google.gax import GaxError
#pylint: enable=no-name-in-module, import-error, relative-import

class SpannerWrapper(object):
    """SpannerWrapper class handles all interactions with cloud Spanner."""
    JOB_CONFIGS_TABLE = "JobConfigs"
    JOB_CONFIG_ID = "JobConfigId"
    JOB_SPEC = "JobSpec"
    JOB_CONFIGS_COLUMNS = [JOB_CONFIG_ID, JOB_SPEC]

    JOB_RUNS_TABLE = "JobRuns"
    JOB_RUN_ID = "JobRunId"
    JOB_CREATION_TIME = "JobCreationTime"
    STATUS = "Status"
    JOB_RUNS_COLUMNS = [JOB_CONFIG_ID, JOB_RUN_ID, STATUS, JOB_CREATION_TIME]

    TASKS_TABLE = "Tasks"
    TASK_ID = "TaskId"
    FAILURE_MESSAGE = "FailureMessage"
    LAST_MODIFICATION_TIME = "LastModificationTime"
    TASK_SPEC = "TaskSpec"
    WORKER_ID = "WorkerId"
    TASKS_COLUMNS = [
        JOB_CONFIG_ID, JOB_RUN_ID, TASK_ID, FAILURE_MESSAGE,
        LAST_MODIFICATION_TIME, STATUS, TASK_SPEC, WORKER_ID
    ]

    # Used to limit the number of rows to avoid OOM errors
    # TODO(b/64092801): Replace cap with streaming of large results
    ROW_CAP = 10000

    def __init__(self, json_key_file_path, instance_id, database_id):
        """Creates and initializes an instance of the SpannerWrapper class.

        Args:
          json_key_file_path: The path to the JSON file holding the key
                              for a Cloud Spanner authorized service account.
          instance_id: The id of the Cloud Spanner instance.
          database_id: The id of the Cloud Spanner instance.
        """
        self.spanner_client = spanner.Client.from_service_account_json(
            json_key_file_path)

        # Get a Cloud Spanner instance by ID.
        self.instance = self.spanner_client.instance(instance_id)

        # Get a Cloud Spanner database by ID.
        self.database = self.instance.database(database_id)

        self.session_pool = spanner.pool.BurstyPool()
        self.session_pool.bind(self.database)

    def get_job_configs(self):
        """Retrieves all job configs from Cloud Spanner.

        Returns:
          A list containing the retrieved job configs in JSON format.
        """
        query = "SELECT * FROM %s" % SpannerWrapper.JOB_CONFIGS_TABLE
        return self.list_query(query)

    def get_job_config(self, config_id):
        """Retrieves the specified job config from Cloud Spanner.

        Args:
          config_id: The id of the desired job config.

        Returns:
          A dictionary containing the desired job config, mapping from
          attribute to value.
        """
        query = ("SELECT * FROM %s WHERE %s = @config_id" %
                 (SpannerWrapper.JOB_CONFIGS_TABLE,
                  SpannerWrapper.JOB_CONFIG_ID))
        return self.single_result_query(
            query,
            {"config_id": config_id},
            {"config_id": type_pb2.Type(code=type_pb2.STRING)}
        )

    def create_job_config(self, config_id, job_spec):
        """Creates a new job config using the given config attributes.

        Args:
          config_id: The desired config id for the new job config
          job_spec: The desired job spec for the new job config

        Returns:
          True if the JobConfig was created, false otherwise.
        """
        config_id = unicode(config_id)
        job_spec = unicode(job_spec)
        values = [config_id, job_spec]

        return self.insert(SpannerWrapper.JOB_CONFIGS_TABLE,
                           SpannerWrapper.JOB_CONFIGS_COLUMNS, values)

    def create_job_run(self, config_id, run_id):
        """Creates a new job run with the given JobRun attributes.

        Args:
          config_id: The desired JobConfigId of the new job run
          run_id: The desired JobRunId of the new job run

        Returns:
          True if the JobRun was created, false otherwise.
        """
        config_id = unicode(config_id)
        run_id = unicode(run_id)
        values = [config_id, run_id, 1, int(time.time())]

        return self.insert(SpannerWrapper.JOB_RUNS_TABLE,
                           SpannerWrapper.JOB_RUNS_COLUMNS, values)

    def get_job_runs(self, max_num_runs=25, created_before=None):
        """Retrieves job runs from Cloud Spanner.

        Retrieves 0 to max_num_runs job runs. If a created_before time is
        specified, only jobs created before the given time will be returned
        (created_before is intended for use as a continuation token for paging).
        The returned job runs are sorted by creation time, with the most
        recent runs listed first.

        Args:
            max_num_runs: 0 to max_num_runs will be returned. Must be > 0.
                          Defaults to 25.
            created_before: The time before which all returned runs were created

        Returns:
          A list of dictionaries, where each dictionary represents a job run.

        Raises:
          ValueError: If max_num_runs is <= 0 or > ROW_CAP
        """
        if max_num_runs <= 0:
            raise ValueError("max_num_runs must be greater than 0")
        if max_num_runs > SpannerWrapper.ROW_CAP:
            raise ValueError("max_num_runs must be less than or equal to %d" %
                             SpannerWrapper.ROW_CAP)

        query = "SELECT * FROM %s" % SpannerWrapper.JOB_RUNS_TABLE
        params = {"num_runs": max_num_runs}
        param_types = {"num_runs": type_pb2.Type(code=type_pb2.INT64)}
        if created_before:
            query += " WHERE %s < @created_before" % (
                SpannerWrapper.JOB_CREATION_TIME)
            params["created_before"] = created_before
            param_types["created_before"] = type_pb2.Type(code=type_pb2.INT64)

        query += " ORDER BY %s DESC LIMIT @num_runs" % (
            SpannerWrapper.JOB_CREATION_TIME)
        return self.list_query(query, params, param_types)

    # pylint: disable=too-many-arguments
    def get_tasks_for_run(self, config_id, run_id, max_num_tasks=25,
                          task_type=None, last_modified=None):
        """Retrieves the tasks with the given type for the specified job run.

        Retrieves the tasks for the specified job run from Cloud Spanner. If
        a task type is specified, only retrieves tasks of that type. Otherwise
        returns alls tasks for the given job run. Tasks are sorted by
        last modification time, with the most recently modified tasks
        listed first.

        Args:
            config_id: The config id of the desired tasks
            run_id: The job run id of the desired tasks
            max_num_tasks: The number of tasks to return, default is 25.
                           Must be > 0. max_num_tasks is the max number of
                           tasks returned, less than max_num_tasks will be
                           returned if there are not enough matching tasks.
            task_type: The desired type of the tasks, defaults to None.
            last_modified: All returned tasks will have a last_modified time
                         less than the given time

        Returns:
          A dictionary containing the tasks matching the given parameters.

        Raises:
          ValueError: If max_num_tasks is <= 0 or > ROW_CAP
        """
        if max_num_tasks <= 0:
            raise ValueError("max_num_tasks must be greater than 0")
        if max_num_tasks > SpannerWrapper.ROW_CAP:
            raise ValueError("max_num_tasks must be less than or equal to %d" %
                             SpannerWrapper.ROW_CAP)

        params = {
            "run_id": run_id,
            "config_id": config_id,
            "num_tasks": max_num_tasks
        }
        param_types = {
            "run_id": type_pb2.Type(code=type_pb2.STRING),
            "config_id": type_pb2.Type(code=type_pb2.STRING),
            "num_tasks": type_pb2.Type(code=type_pb2.INT64)
        }
        query = ("SELECT * FROM %s WHERE %s = @run_id AND %s = @config_id" %
                 (SpannerWrapper.TASKS_TABLE, SpannerWrapper.JOB_RUN_ID,
                  SpannerWrapper.JOB_CONFIG_ID))

        if last_modified:
            query += (" AND %s < @last_modified" %
                      SpannerWrapper.LAST_MODIFICATION_TIME)
            params["last_modified"] = last_modified
            param_types["last_modified"] = type_pb2.Type(code=type_pb2.INT64)
        if task_type:
            query += (
                " AND STARTS_WITH(%s, @task_type)" % SpannerWrapper.TASK_ID)
            params["task_type"] = task_type
            param_types["task_type"] = type_pb2.Type(code=type_pb2.STRING)
        query += (" ORDER BY %s DESC LIMIT @num_tasks" %
                  SpannerWrapper.LAST_MODIFICATION_TIME)
        return self.list_query(query, params, param_types)

    # pylint: enable=too-many-arguments
    def get_job_run(self, config_id, run_id):
        """Retrieves the job run with the specified job run id.

        Args:
          config_id: The config id of the desired job run.
          run_id: The job run id of the desired job run.

        Returns:
          A dictionary containing the job_run with the given job_run_id.
        """
        query = ("SELECT * FROM %s WHERE %s = @run_id AND %s = @config_id" %
                 (SpannerWrapper.JOB_RUNS_TABLE, SpannerWrapper.JOB_RUN_ID,
                  SpannerWrapper.JOB_CONFIG_ID))
        return self.single_result_query(
            query,
            {"run_id": run_id, "config_id": config_id},
            {
                "run_id": type_pb2.Type(code=type_pb2.STRING),
                "config_id": type_pb2.Type(code=type_pb2.STRING)
            }
        )

    def insert(self, table, columns, values):
        """Inserts the given values into the specified table.

        Args:
          table: The name of the table for the insertion
          columns: The columns of the values, passed as an array of strings.
          values: The values to insert into the given columns. Passed as an
                  array. Note: Any string values should be in unicode.

        Returns:
          True if the insertion succeeds, False otherwise.
        """
        with self.session_pool.session() as session:
            try:
                with session.transaction() as transaction:
                    transaction.insert(table, columns=columns, values=[values])
                    return True
            except GaxError:
                # TODO(b/64075962): Better error handling
                logging.exception("Error inserting into Cloud Spanner")
                return False

    def list_query(self, query, query_params=None, param_types=None):
        """Performs the given query and processes the result list.

        Performs the given query and processes the result list, forming
        a list of objects that map from attribute to value. One object
        corresponds to on result row.

        Args:
          query: The query to be performed, with parameters in the form @name.
                 Example "SELECT * FROM Table WHERE attribute = @name"
          query_params: Any parameters used in the query (dict name->value)
          param_types: The types of any parameters used in the query
                       (dict name->TypeCode)
                       Example {"name": type_pb2.Type(code=type_pb2.STRING)}

        Returns:
          A list of dictionaries mapping from attribute name to value
        """
        results = self.database.execute_sql(query, query_params, param_types)
        result_list = []
        for row in results:
            obj = self.row_to_object(row, results.fields)
            result_list.append(obj)

        return result_list

    def single_result_query(self, query, query_params=None, param_types=None):
        """Performs the given query and processes the result.

        Performs the given query and processes the result, returning an object
        that maps from attribute to value. If the query returns more than one
        result, only the first row will be processed and returned.

        Args:
          query: The query to be performed, with parameters in the form @name
                 Example "SELECT * FROM Table WHERE attribute = @name"
          query_params: Any parameters used in the query (dict name->value)
          param_types: The types of any parameters used in the query
                       (dict name->TypeCode)
                       Example {"name": type_pb2.Type(code=type_pb2.STRING)}

        Returns:
          A dictionary mapping from attribute name to value, or None if the
          query had no results.
        """
        results = self.database.execute_sql(query, query_params, param_types)

        for row in results:
            return self.row_to_object(row, results.fields)

    @staticmethod
    def row_to_object(result, fields):
        """Processes a single result of a StreamedResultSet and returns it.

        Processes a single result of a StreamedResultSet and returns an object
        that maps from attribute name to value.

        Args:
          result: A single row of a StreamedResultSet
          fields: The fields of the result

        Returns:
          A dictionary mapping from attribute name to value
        """
        obj = {}
        for i, field in enumerate(fields):
            obj[field.name] = result[i]
            i += 1

        return obj
