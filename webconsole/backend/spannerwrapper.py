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
import json
import os
import time
import util
from google.cloud import spanner
from google.cloud.spanner_v1.proto import type_pb2
from gaxerrordecorator import handle_common_gax_errors

from create_infra.job_utilities import TASK_STATUS_UNQUEUED
from create_infra.job_utilities import TASK_TYPE_LIST
from create_infra.job_utilities import JOB_STATUS_IN_PROGRESS
from proto.tasks_pb2 import TaskFailureType
from proto.tasks_pb2 import TaskStatus
from proto.tasks_pb2 import TaskType

from google.cloud.exceptions import BadRequest

def _check_max_num_tasks_in_range(max_num_tasks):
    """Checks that the maximum number of tasks is in the allowed range.
    Args:
        max_num_tasks: The number of tasks.
    Raises:
        BadRequest: If max_num_tasks is <= 0 or > ROW_CAP
    """
    if max_num_tasks <= 0 or max_num_tasks > SpannerWrapper.ROW_CAP:
        raise BadRequest("max_num_tasks must be a number between 1 and %d" %
                         SpannerWrapper.ROW_CAP)

def _get_tasks_params_and_types(config_id, run_id, max_num_tasks,
    task_status=None, last_modified_before=None, failure_type=None):
    """Gets the base parameters and parameter types used in most queries.
    Returns:
        A tuple of params and param_types of the specified input parameters.
    """
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
    if task_status is not None:
        params["task_status"] = task_status
        param_types["task_status"] = type_pb2.Type(code=type_pb2.INT64)
    if failure_type is not None:
        params["failure_type"] = failure_type
        param_types["failure_type"] = type_pb2.Type(code=type_pb2.INT64)
    if last_modified_before is not None:
        params["last_modified_before"] = last_modified_before
        param_types["last_modified_before"] = type_pb2.Type(code=type_pb2.INT64)
    return params, param_types

def _get_delible_indelible_configs(config_id_list, rows):
    """Gets a list of delible and indelible configs from the result of the
       union query to get tasks in progress.
    """
    delible_configs = []
    indelible_configs = []
    for i, row in enumerate(rows):
        tasks_count = row[0]
        if tasks_count > 0:
            indelible_configs.append(config_id_list[i])
        else:
            delible_configs.append(config_id_list[i])
    return delible_configs, indelible_configs

def _get_all_tasks_in_progress(config_id_list):
    """Gets a string query that returns the number of tasks in progress for
       each config in config_id_list.

    Args:
        config_id_list: The list of job config ids to build the query from.
    Returns:
        1) A string query that has one row per config_id_list. Each row has the
        count of the tasks in progress for each config in config_id_list.
        2) The params for the query
        3) The param_types for the query
    """
    params = {}
    param_types = {}
    query = ''
    for i, config_id in enumerate(config_id_list):
        query += _get_tasks_in_progress_query(config_id, params, param_types, i)
        if i < len(config_id_list)-1:
            query = query + '\nUNION ALL\n'
    return query, params, param_types

def _delete_job_configs_transaction(transaction, config_id_list,
    transaction_read_result):
    """Reads the tasks in each job configuration, and deletes a job
       configuration if it has no tasks in progress.

    Args:
        transaction: The transaction to operate with.
        config_id_list: The list of job configurations to try to delete.
        transaction_read_result: A dictionary with fields 'delible_configs' and
            'indelible_configs'. The function populates these fields with what
            it finds out about reading the tasks for the job configs.
    Returns:
        No return value.
    Raises:
        ValueError if there are no job configurations to delete.
    """
    query, params, param_types = \
        _get_all_tasks_in_progress(config_id_list)
    result = \
        transaction.execute_sql(query, params=params, param_types=param_types)
    result.consume_all()
    delible_configs, indelible_configs = \
        _get_delible_indelible_configs(config_id_list, result.rows)
    transaction_read_result['delible_configs'] = delible_configs
    transaction_read_result['indelible_configs'] = indelible_configs
    if delible_configs:
        # Put keys in a list of lists, the expected keyset format.
        keyset_keys = [[config_id] for config_id in delible_configs]
        transaction.delete(SpannerWrapper.JOB_CONFIGS_TABLE,
                            keyset=spanner.KeySet(keys=keyset_keys))

def _get_tasks_in_progress_query(config_id, params, param_types, counter):
    """Gets a string query that will return the number of tasks that are in
        progress.

    Args:
        config_id: The config id to use for the query.
        counter: The config counter. This is used to populate params and
          param_types.
        params: The function populates this dictionary with key config_<counter>
          and value equal to config_id.
        param_types: The function populates this dictionary with key
          config_<counter> and value equal to string parameter type.

    Returns:
        A string query that counts the number of tasks that are in a QUEUED or
        UNQUEUED state. The name of the column given to this count is
        'tasks_in_progress_count'
    """
    config_id_param = '@config_' + str(counter)
    config_id_key = 'config_' + str(counter)
    query = ('SELECT COUNT(*) AS tasks_in_progress_count FROM {0} '
            'WHERE {1} = {2} AND  ({3} = {4} OR {3} = {5})').format(
            SpannerWrapper.TASKS_TABLE, SpannerWrapper.JOB_CONFIG_ID,
            config_id_param, SpannerWrapper.STATUS, TaskStatus.QUEUED,
            TaskStatus.UNQUEUED)
    params[config_id_key] = config_id
    param_types[config_id_key] = type_pb2.Type(code=type_pb2.STRING)
    return query

def _get_tasks_of_status_base_query():
    """Gets the base query to get the task status.
    """
    return (("SELECT * FROM %s@{FORCE_INDEX=%s} WHERE %s = @config_id "
             "AND %s = @run_id AND %s = @task_status ") %
            (SpannerWrapper.TASKS_TABLE,
             SpannerWrapper.TASK_BY_STATUS_INDEX_NAME,
             SpannerWrapper.JOB_CONFIG_ID,
             SpannerWrapper.JOB_RUN_ID,
             SpannerWrapper.STATUS))

class SpannerWrapper(object):
    """SpannerWrapper class handles all interactions with cloud Spanner.

    Any of the methods in the class can raise the following exceptions:
        Forbidden - Not allowed to access the specified Project or Spanner
                    resources
        NotFound - Allowed to access the Spanner resource, but it doesn't exist
        Unauthorized - Not properly authorized
    """
    JOB_CONFIGS_TABLE = "JobConfigs"
    JOB_CONFIG_ID = "JobConfigId"
    JOB_SPEC = "JobSpec"
    JOB_CONFIGS_COLUMNS = [JOB_CONFIG_ID, JOB_SPEC]

    JOB_RUNS_TABLE = "JobRuns"
    JOB_RUN_ID = "JobRunId"
    JOB_CREATION_TIME = "JobCreationTime"
    STATUS = "Status"
    COUNTERS = "Counters"
    JOB_RUNS_COLUMNS = [JOB_CONFIG_ID, JOB_RUN_ID, STATUS, JOB_CREATION_TIME,
                        COUNTERS]

    TASKS_TABLE = "Tasks"
    TASK_ID = "TaskId"
    TASK_CREATION_TIME = "CreationTime"
    LAST_MODIFICATION_TIME = "LastModificationTime"
    TASK_SPEC = "TaskSpec"
    TASK_TYPE = "TaskType"
    FAILURE_MESSAGE = "FailureMessage"
    WORKER_ID = "WorkerId"
    FAILURE_TYPE = "FailureType"

    TASK_BY_STATUS_INDEX_NAME = "TasksByStatus"
    BY_FAILURE_TYPE_INDEX_NAME = "TasksByFailureType"
    TASKS_COLUMNS = [
        JOB_CONFIG_ID, JOB_RUN_ID, TASK_ID, TASK_CREATION_TIME,
        LAST_MODIFICATION_TIME, STATUS, TASK_SPEC, TASK_TYPE]

    # Used to limit the number of rows to avoid OOM errors
    # TODO(b/64092801): Replace cap with streaming of large results
    ROW_CAP = 10000

    @handle_common_gax_errors
    def __init__(self, credentials, project_id, instance_id, database_id):
        """Creates and initializes an instance of the SpannerWrapper class.

        Args:
            credentials: The OAuth2 Credentials to use to create spanner
                         instance.
            project_id: The cloud ingest project id.
            instance_id: The id of the Cloud Spanner instance.
            database_id: The id of the Cloud Spanner instance.
        """
        self.project_id = project_id
        self.instance_id = instance_id
        self.database_id = database_id
        self.spanner_client = spanner.Client(credentials=credentials,
                                             project=project_id)
        # Get a Cloud Spanner instance by ID.
        self.instance = self.spanner_client.instance(instance_id)

        # Get a Cloud Spanner database by ID.
        self.database = self.instance.database(database_id)

        self.session_pool = spanner.BurstyPool()
        self.session_pool.bind(self.database)

    def get_job_configs(self):
        """Retrieves all job configs from Cloud Spanner.

        Returns:
            A list containing the retrieved job configs in JSON format.
        """
        query = "SELECT * FROM %s" % SpannerWrapper.JOB_CONFIGS_TABLE
        list_query = self.list_query(query)
        return util.json_to_dictionary_in_field(list_query, self.JOB_SPEC)

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

        Raises:
            Conflict if the job config already exists
        """
        config_id = unicode(config_id)
        job_spec = unicode(job_spec)
        values = [config_id, job_spec]

        self.insert(SpannerWrapper.JOB_CONFIGS_TABLE,
                    SpannerWrapper.JOB_CONFIGS_COLUMNS, values)

    def create_job_run(self, config_id, run_id, initial_total_tasks=0):
        """Creates a new job run with the given JobRun attributes.

        Args:
            config_id: The desired JobConfigId of the new job run
            run_id: The desired JobRunId of the new job run
            initial_total_tasks: Initial number of total tasks in the job run.

        Raises:
            Conflict if the job run already exists
        """
        # TODO(b/65943019): Remove initial_total_tasks params. This should
        # be always 0. This param should be removed after the DCP has proper
        # handling of job scheduling.
        config_id = unicode(config_id)
        run_id = unicode(run_id)
        counters = {
            # Overall job run stats.
            'totalTasks': initial_total_tasks,
            'tasksCompleted': 0,
            'tasksFailed': 0,
            'tasksQueued': 0,
            'tasksUnqueued': initial_total_tasks,

            # List task stats.
            'totalTasksList': initial_total_tasks,
            'tasksCompletedList': 0,
            'tasksFailedList': 0,
            'tasksQueuedList': 0,
            'tasksUnqueuedList': initial_total_tasks,

            # Copy task stats.
            'totalTasksCopy': 0,
            'tasksCompletedCopy': 0,
            'tasksFailedCopy': 0,
            'tasksQueuedCopy': 0,
            'tasksUnqueuedCopy': 0
        }
        # The job status is set to in counters because the first list
        # task is manually inserted. When the logic in the DCP is changed,
        # new jobs should be inserted with a status of not started.
        values = [config_id, run_id, JOB_STATUS_IN_PROGRESS,
                  self._get_unix_nano(), json.dumps(counters)]

        self.insert(SpannerWrapper.JOB_RUNS_TABLE,
                    SpannerWrapper.JOB_RUNS_COLUMNS, values)

    def create_job_run_first_list_task(self, config_id, run_id, task_id,
                                       job_spec):
        """DO NOT USE, only intended for temporary use by flask app job_configs
        handler method in main.py. Creates the first listing task for a job run.

        TODO(b/65846311): The web console should not schedule the job runs and
        should not create the first task. Remove this method after the
        functionality is added to the DCP.
        """
        config_id = unicode(config_id)
        run_id = unicode(run_id)
        task_id = unicode(task_id)

        job_spec_dict = json.loads(job_spec)

        if 'gcsDirectory' in job_spec_dict:
            list_result_object_name = os.path.join(
            job_spec_dict['gcsDirectory'],
            'list-task-output-%s-%s-%s' % (config_id,
                                           run_id,
                                           task_id))
        else:
            list_result_object_name = 'list-task-output-%s-%s-%s' % (config_id,
                                                                     run_id,
                                                                     task_id)
        task_spec = {
            'src_directory': job_spec_dict['onPremSrcDirectory'],
            'dst_list_result_bucket': job_spec_dict['gcsBucket'],
            'dst_list_result_object': list_result_object_name,
            'expected_generation_num': 0,
        }

        current_time_nanos = self._get_unix_nano()

        values = [
            config_id,
            run_id,
            task_id,
            current_time_nanos,
            current_time_nanos,
            TASK_STATUS_UNQUEUED,
            json.dumps(task_spec).encode('utf-8'),
            TASK_TYPE_LIST,
        ]

        self.insert(SpannerWrapper.TASKS_TABLE,
                    SpannerWrapper.TASKS_COLUMNS, values)

    def get_job_runs(self, max_num_runs, created_before=None):
        """Retrieves job runs from Cloud Spanner.

        Retrieves 0 to max_num_runs job runs. If a created_before time is
        specified, only jobs created before the given time will be returned
        (created_before is intended for use as a continuation token for paging).
        The returned job runs are sorted by creation time, with the most
        recent runs listed first.

        Args:
            max_num_runs: 0 to max_num_runs will be returned. Must be > 0
                          and < ROW_CAP.
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
        job_runs = self.list_query(query, params, param_types)
        return util.json_to_dictionary_in_field(job_runs,
                                                SpannerWrapper.COUNTERS)

    # pylint: disable=too-many-arguments
    def get_tasks_for_run(self, config_id, run_id, max_num_tasks,
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
            max_num_tasks: The number of tasks to return. Must be > 0.
                           max_num_tasks is the max number of tasks returned,
                           less than max_num_tasks will be returned if there
                           are not enough matching tasks.
            task_type: The desired type of the tasks, defaults to None.
            last_modified: All returned tasks will have a last_modified time
                         less than the given time

        Returns:
          A dictionary containing the tasks matching the given parameters.

        Raises:
          BadRequest: If max_num_tasks is not in range.
        """
        _check_max_num_tasks_in_range(max_num_tasks)
        if (task_type is not None and
            task_type not in TaskType.Type.values()):
            raise BadRequest("Task of type %d is unknown.", task_type)
        params, param_types = _get_tasks_params_and_types(config_id,
            run_id, max_num_tasks)

        query = ("SELECT * FROM %s WHERE %s = @config_id AND %s = @run_id " %
            (SpannerWrapper.TASKS_TABLE, SpannerWrapper.JOB_CONFIG_ID,
                SpannerWrapper.JOB_RUN_ID))
        if last_modified:
            params["last_modified"] = last_modified
            param_types["last_modified"] = type_pb2.Type(code=type_pb2.INT64)
            query += ("AND %s < @last_modified " %
                SpannerWrapper.LAST_MODIFICATION_TIME)
        if task_type:
            params["task_type"] = task_type
            param_types["task_type"] = type_pb2.Type(code=type_pb2.INT64)
            query += ("AND %s = @task_type " % SpannerWrapper.TASK_TYPE)
        query += ("ORDER BY %s DESC LIMIT @num_tasks" %
                  SpannerWrapper.LAST_MODIFICATION_TIME)
        return self.list_query(query, params, param_types)

    def get_tasks_of_status(self, config_id, run_id, max_num_tasks,
                            task_status, last_modified_before=None):
        """Retrieves tasks of the input status for the input job configuration
           and job run.

        Args:
          config_id: The job configuration id.
          run_id: The job run id.
          max_num_tasks: The maximum number of tasks to retrieve.
          task_status: Get tasks of this status.
          last_modified_before: Retrieves tasks only before this timestamp.

        Raises:
          BadRequest: if max_num_tasks is not in the allowed range or if the
                      task status is not in the allowed range.

        """
        _check_max_num_tasks_in_range(max_num_tasks)
        if task_status not in TaskStatus.Type.values():
            raise BadRequest("Task status of id %d is unknown.", task_status)
        params, param_types = _get_tasks_params_and_types(config_id,
            run_id, max_num_tasks, task_status=task_status,
            last_modified_before=last_modified_before)
        query = _get_tasks_of_status_base_query()
        if last_modified_before is not None:
            query += ("AND %s < @last_modified_before " %
                SpannerWrapper.LAST_MODIFICATION_TIME)
        query += ("ORDER BY %s DESC LIMIT @num_tasks" %
            SpannerWrapper.LAST_MODIFICATION_TIME)
        return self.list_query(query, params, param_types)

    def get_tasks_of_failure_type(self, config_id, run_id, max_num_tasks,
                                  failure_type, last_modified_before=None):
        """Retrieves the tasks of the input failure type for the input job
           configuration and job run.

        Args:
          config_id: The job configuration id.
          run_id: The job run id.
          max_num_tasks: The maximum number of tasks to retrieve.
          failure_type: Get tasks of this failure type.

        Raises:
          BadRequest: if max_num_tasks is not in the allowed range or if the
                      failure type is not in the allowed range.
        """
        _check_max_num_tasks_in_range(max_num_tasks)
        if failure_type not in TaskFailureType.Type.values():
            raise BadRequest("Task of failure type %d is unknown.",
                failure_type)
        params, param_types = _get_tasks_params_and_types(config_id,
            run_id, max_num_tasks, failure_type=failure_type,
            last_modified_before=last_modified_before)

        query = (("SELECT * FROM %s@{FORCE_INDEX=%s} WHERE %s = @config_id "
                  "AND %s = @run_id AND %s = @failure_type ") %
                (SpannerWrapper.TASKS_TABLE,
                 SpannerWrapper.TASK_BY_STATUS_INDEX_NAME,
                 SpannerWrapper.JOB_CONFIG_ID,
                 SpannerWrapper.JOB_RUN_ID,
                 SpannerWrapper.FAILURE_TYPE))

        if last_modified_before is not None:
            query += ("AND %s < @last_modified_before " %
                      SpannerWrapper.LAST_MODIFICATION_TIME)
        query += ("ORDER BY %s DESC LIMIT @num_tasks"
            % SpannerWrapper.LAST_MODIFICATION_TIME)
        return self.list_query(query, params, param_types)

    # pylint: enable=too-many-arguments
    def get_job_run(self, config_id, run_id):
        """Retrieves the job run with the specified job run id.

        Args:
          config_id: The config id of the desired job run.
          run_id: The job run id of the desired job run.

        Returns:
          A dictionary containing the job_run with the given job_run_id or
          None if no such job run exists.
        """
        query = ("SELECT * FROM %s WHERE %s = @run_id AND %s = @config_id" %
                 (SpannerWrapper.JOB_RUNS_TABLE, SpannerWrapper.JOB_RUN_ID,
                  SpannerWrapper.JOB_CONFIG_ID))
        job_run = self.single_result_query(
            query,
            {"run_id": run_id, "config_id": config_id},
            {
                "run_id": type_pb2.Type(code=type_pb2.STRING),
                "config_id": type_pb2.Type(code=type_pb2.STRING)
            }
        )
        if job_run:
            job_run[SpannerWrapper.COUNTERS] = json.loads(
                job_run[SpannerWrapper.COUNTERS])
            return job_run

    def delete_job_configs(self, config_id_list):
        """Deletes a list of job configurations. Only deletes a job
           configuration if it doesn't have tasks in progress.

        Args:
          config_id_list: The list of job config ids to delete.
        Returns:
          A list of configs that could not be deleted because they had tasks in
          progress, or None if all the input configs were deleted.
        """
        transaction_read_result = {
            'delible_configs': [],
            'indelible_configs': []
        }
        try:
            self.database.run_in_transaction(_delete_job_configs_transaction,
                config_id_list, transaction_read_result)
        except ValueError:
            pass # There was nothing to commit. Do nothing.
        return transaction_read_result

    @handle_common_gax_errors
    def insert(self, table, columns, values):
        """Inserts the given values into the specified table.

        Args:
          table: The name of the table for the insertion
          columns: The columns of the values, passed as an array of strings.
          values: The values to insert into the given columns. Passed as an
                  array. Note: Any string values should be in unicode.

        Raises:
          Conflict: If the item to insert already exists
        """
        with self.session_pool.session() as session:
            with session.transaction() as transaction:
                transaction.insert(table, columns=columns,
                                   values=[values])

    @handle_common_gax_errors
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
        result_list = []
        with self.database.snapshot() as snapshot:
            results = snapshot.execute_sql(query, query_params, param_types)
            for row in results:
                obj = self.row_to_object(row, results.fields)
                result_list.append(obj)
        return result_list

    @handle_common_gax_errors
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
        with self.database.snapshot() as snapshot:
            results = snapshot.execute_sql(query, query_params, param_types)

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

    @staticmethod
    def _get_unix_nano():
        """Returns the current Unix time in nanoseconds

        Returns:
            An integer representing the current Unix time in nanoseconds
        """
        # time.time() returns Unix time in seconds. Multiply by 1e9 to get
        # the time in nanoseconds
        return int(time.time() * 1e9)
