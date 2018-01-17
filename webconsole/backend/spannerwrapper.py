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

from google.cloud import spanner
from google.cloud.spanner_v1.proto import type_pb2

import util
from gaxerrordecorator import handle_common_gax_errors
from proto.tasks_pb2 import TaskStatus
from proto.tasks_pb2 import TaskType
from proto.tasks_pb2 import JobRunStatus

# pylint: disable=too-many-arguments
def _get_params_and_param_types(project_id=None, config_id=None, run_id=None,
    num_tasks=None, task_status=None, last_modified_before=None,
    failure_type=None):
    """Gets the base parameters and parameter types used in most queries.

    Returns:
        A tuple of params and param_types of the specified input parameters.
    """
    params = {
    }
    param_types = {
    }
    if project_id is not None:
        params["project_id"] = project_id
        param_types["project_id"] = type_pb2.Type(code=type_pb2.STRING)
    if config_id is not None:
        params["config_id"] = config_id
        param_types["config_id"] = type_pb2.Type(code=type_pb2.STRING)
    if run_id is not None:
        params["run_id"] = run_id
        param_types["run_id"] = type_pb2.Type(code=type_pb2.STRING)
    if num_tasks is not None:
        params["num_tasks"] = num_tasks
        param_types["task_status"] = type_pb2.Type(code=type_pb2.INT64)
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
# pylint: enable=too-many-arguments

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

def _get_all_tasks_in_progress(project_id, config_id_list):
    """Gets a string query that returns the number of tasks in progress for
       each config in config_id_list.

    Args:
        project_id: The project id used in the query.
        config_id_list: The list of job config ids to build the query from.
    Returns:
        1) A string query that has one row per config_id_list. Each row has the
        count of the tasks in progress for each config in config_id_list.
        2) The params for the query
        3) The param_types for the query
    """
    params, param_types = _get_params_and_param_types(project_id=project_id)
    query = ''
    for i, config_id in enumerate(config_id_list):
        query += _get_tasks_in_progress_query(config_id, params, param_types, i)
        if i < len(config_id_list)-1:
            query = query + '\nUNION ALL\n'
    return query, params, param_types

def _delete_job_configs_transaction(transaction, project_id, config_id_list,
    transaction_read_result):
    """Reads the tasks in each job configuration, and deletes a job
       configuration if it has no tasks in progress.

    Args:
        transaction: The transaction to operate with.
        project_id: The project id of the configs to delete.
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
        _get_all_tasks_in_progress(project_id, config_id_list)
    result = \
        transaction.execute_sql(query, params=params, param_types=param_types)
    result.consume_all()
    delible_configs, indelible_configs = \
        _get_delible_indelible_configs(config_id_list, result.rows)
    transaction_read_result['delible_configs'] = delible_configs
    transaction_read_result['indelible_configs'] = indelible_configs
    if delible_configs:
        # Put keys in a list of lists, the expected keyset format.
        keyset_keys = [[project_id, config_id] for config_id in delible_configs]
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
    query = """
        SELECT COUNT(*) AS tasks_in_progress_count FROM %(tasks_table)s
        WHERE %(project_id_col)s = @project_id AND
            %(config_id_col)s = %(config_id_param)s AND
            (
                %(status_col)s = %(status_queued)d OR
                %(status_col)s = %(status_unqueued)d
            )
        """ % {
            'tasks_table': SpannerWrapper.TASKS_TABLE,
            'project_id_col': SpannerWrapper.PROJECT_ID,
            'config_id_col': SpannerWrapper.JOB_CONFIG_ID,
            'config_id_param': config_id_param,
            'status_col': SpannerWrapper.STATUS,
            'status_queued': TaskStatus.QUEUED,
            'status_unqueued': TaskStatus.UNQUEUED
        }
    params[config_id_key] = config_id
    param_types[config_id_key] = type_pb2.Type(code=type_pb2.STRING)
    return query

def _get_tasks_of_status_base_query():
    """Gets the base query to get the task status.
    """
    return (("SELECT * FROM %s@{FORCE_INDEX=%s} WHERE %s = @project_id "
             "AND %s = @config_id AND %s = @run_id AND %s = @task_status ") %
            (SpannerWrapper.TASKS_TABLE,
             SpannerWrapper.TASK_BY_STATUS_INDEX_NAME,
             SpannerWrapper.PROJECT_ID,
             SpannerWrapper.JOB_CONFIG_ID,
             SpannerWrapper.JOB_RUN_ID,
             SpannerWrapper.STATUS))

# The number of task rows to retrieve per query.
_NUM_OF_TASKS = 25

# TODO(b/65846311): Temporary constants used to create the job run and first
# list task when a job config is created. Eventually DCP should manage
# scheduling/creating job runs and first listing tasks.
_FIRST_JOB_RUN_ID = "jobrun"
_LIST_TASK_ID = "list"

_INITIAL_TOTAL_TASKS = 1
_INITIAL_COUNTERS_DICT = {
    # Overall job run stats.
    'totalTasks': _INITIAL_TOTAL_TASKS,
    'tasksCompleted': 0,
    'tasksFailed': 0,
    'tasksQueued': 0,
    'tasksUnqueued': _INITIAL_TOTAL_TASKS,

    # List task stats.
    'totalTasksList': _INITIAL_TOTAL_TASKS,
    'tasksCompletedList': 0,
    'tasksFailedList': 0,
    'tasksQueuedList': 0,
    'tasksUnqueuedList': _INITIAL_TOTAL_TASKS,

    # Copy task stats.
    'totalTasksCopy': 0,
    'tasksCompletedCopy': 0,
    'tasksFailedCopy': 0,
    'tasksQueuedCopy': 0,
    'tasksUnqueuedCopy': 0
}

def _get_task_spec_first_list_task(job_spec_dict, config_id):
    """
    Gets the task spec for the first list task from a job spec dictionary.
    """
    list_result_object_name = 'cloud-ingest/listfiles/%s/%s/%s' % (config_id,
        _FIRST_JOB_RUN_ID, 'list')
    task_spec = {
        'src_directory': job_spec_dict['onPremSrcDirectory'],
        'dst_list_result_bucket': job_spec_dict['gcsBucket'],
        'dst_list_result_object': list_result_object_name,
        'expected_generation_num': 0,
    }
    return task_spec

# TODO(b/65943019): The web console should not schedule the job runs and
# should not create the first task. Remove that after the functionality
# is added to the DCP.
def _create_new_job_transaction(transaction, project_id, config_id, job_spec):
    """Creates a new job on the JobConfigs, JobRuns table and inserts the first
       list task on the tasks table.

    Args:
        transaction: The transaction to operate with.
        config_id: The id to give to the job.
        jobspec: The job spec as a dictionary.
    """
    run_id = _FIRST_JOB_RUN_ID
    list_id = 'list:' + job_spec['onPremSrcDirectory']
    task_spec_json = json.dumps(_get_task_spec_first_list_task(job_spec,
        config_id))
    job_spec_json = json.dumps(job_spec)
    current_time_nanos = util.get_unix_nano()
    transaction.insert(SpannerWrapper.JOB_CONFIGS_TABLE,
        columns=SpannerWrapper.JOB_CONFIGS_COLUMNS,
        values=[(project_id, config_id, job_spec_json)])
    # The job status is set to in counters because the first list
    # task is manually inserted. When the logic in the DCP is changed,
    # new jobs should be inserted with a status of not started.
    transaction.insert(SpannerWrapper.JOB_RUNS_TABLE,
        columns=SpannerWrapper.JOB_RUNS_COLUMNS,
        values=[(project_id, config_id, run_id, JobRunStatus.IN_PROGRESS,
        current_time_nanos, json.dumps(_INITIAL_COUNTERS_DICT))])
    transaction.insert(SpannerWrapper.TASKS_TABLE,
        columns=SpannerWrapper.TASKS_COLUMNS, values=[(project_id, config_id,
        run_id, list_id, current_time_nanos, current_time_nanos,
        TaskStatus.UNQUEUED, task_spec_json, TaskType.LIST)])

class SpannerWrapper(object):
    """SpannerWrapper class handles all interactions with cloud Spanner.

    Any of the methods in the class can raise the following exceptions:
        Forbidden - Not allowed to access the specified Project or Spanner
                    resources
        NotFound - Allowed to access the Spanner resource, but it doesn't exist
        Unauthorized - Not properly authorized
    """
    JOB_CONFIGS_TABLE = "JobConfigs"
    PROJECT_ID = "ProjectId"
    JOB_CONFIG_ID = "JobConfigId"
    JOB_SPEC = "JobSpec"
    JOB_CONFIGS_COLUMNS = [PROJECT_ID, JOB_CONFIG_ID, JOB_SPEC]

    JOB_RUNS_TABLE = "JobRuns"
    JOB_RUN_ID = "JobRunId"
    JOB_CREATION_TIME = "JobCreationTime"
    STATUS = "Status"
    COUNTERS = "Counters"
    JOB_RUNS_COLUMNS = [PROJECT_ID, JOB_CONFIG_ID, JOB_RUN_ID, STATUS,
                        JOB_CREATION_TIME, COUNTERS]

    TASKS_TABLE = "Tasks"
    TASK_ID = "TaskId"
    TASK_CREATION_TIME = "CreationTime"
    LAST_MODIFICATION_TIME = "LastModificationTime"
    TASK_SPEC = "TaskSpec"
    TASK_TYPE = "TaskType"
    FAILURE_MESSAGE = "FailureMessage"
    FAILURE_TYPE = "FailureType"

    # The field names that are expected to be converted to json.
    JSON_FIELDS = [JOB_SPEC, COUNTERS, TASK_SPEC]

    TASK_BY_STATUS_INDEX_NAME = "TasksByStatus"
    BY_FAILURE_TYPE_INDEX_NAME = "TasksByFailureType"
    TASKS_COLUMNS = [
        PROJECT_ID, JOB_CONFIG_ID, JOB_RUN_ID, TASK_ID, TASK_CREATION_TIME,
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
            project_id: The cloud ingest project id hosting Spanner instance.
            instance_id: The id of the Cloud Spanner instance.
            database_id: The id of the Cloud Spanner instance.
        """
        self.spanner_client = spanner.Client(credentials=credentials,
                                             project=project_id)
        # Get a Cloud Spanner instance by ID.
        instance = self.spanner_client.instance(instance_id)

        # Get a Cloud Spanner database by ID.
        self.database = instance.database(database_id)

        self.session_pool = spanner.BurstyPool()
        self.session_pool.bind(self.database)

    def get_job_configs(self, project_id):
        """Retrieves all job configs from Cloud Spanner, along with the
           information of the latest job run.

        Args:
            project_id: The project id to get the job configs.

        Returns:
            A list containing the retrieved job configs in JSON format.
        """
        # This query assumes that the job creation time will be unique. If the
        # job creation time is not unique, two rows might be returned with the
        # same config id.
        query = """
            SELECT * FROM %(runs_table)s JOIN %(configs_table)s 
            ON %(runs_table)s.%(project_id_col)s = @project_id AND
                %(runs_table)s.%(project_id_col)s = 
                %(configs_table)s.%(project_id_col)s AND 
                %(runs_table)s.%(config_id_col)s = 
                %(configs_table)s.%(config_id_col)s 
            WHERE %(creation_time_col)s IN (
                SELECT MAX(%(creation_time_col)s) FROM %(runs_table)s 
                WHERE %(runs_table)s.%(project_id_col)s = @project_id
                GROUP BY %(config_id_col)s
            )
            """ % {
            'runs_table': SpannerWrapper.JOB_RUNS_TABLE,
            'configs_table': SpannerWrapper.JOB_CONFIGS_TABLE,
            'project_id_col': SpannerWrapper.PROJECT_ID,
            'config_id_col': SpannerWrapper.JOB_CONFIG_ID,
            'creation_time_col': SpannerWrapper.JOB_CREATION_TIME
        }

        params, param_types = _get_params_and_param_types(project_id=project_id)

        return self.list_query(query, params, param_types)

    def create_new_job(self, project_id, config_id, jobspec):
        """
        Transactionally creates a new job inserting the new job config, the new
        job run, and a the first list task.

        Arguments
            project_id: The project id to use for the job.
            config_id: The configuration id to use for the job.
            job_spec: A dictionary representing the job spec to use for the job.
        """
        self.database.run_in_transaction(_create_new_job_transaction,
            project_id, config_id, jobspec)

    def get_tasks_of_status(self, project_id, config_id, task_status,
        last_modified_before=None):
        """Retrieves tasks of the input status for the input job configuration
           and job run.

        Args:
          project_id: The project id.
          config_id: The job configuration id.
          task_status: Get tasks of this status.
          last_modified_before: Retrieves tasks only before this timestamp.
        """
        params, param_types = _get_params_and_param_types(project_id=project_id,
            config_id=config_id,
            run_id=_FIRST_JOB_RUN_ID,
            num_tasks=_NUM_OF_TASKS,
            task_status=task_status,
            last_modified_before=last_modified_before)
        query = _get_tasks_of_status_base_query()
        if last_modified_before is not None:
            query += ("AND %s < @last_modified_before " %
                SpannerWrapper.LAST_MODIFICATION_TIME)
        query += ("ORDER BY %s DESC LIMIT @num_tasks" %
            SpannerWrapper.LAST_MODIFICATION_TIME)
        return self.list_query(query, params, param_types)

    def get_tasks_of_failure_type(self, project_id, config_id, failure_type,
        last_modified_before=None):
        """Retrieves the tasks of the input failure type for the input job
           configuration and job run.

        Args:
          project_id: Th project id.
          config_id: The job configuration id.
          failure_type: Get tasks of this failure type.
          last_modified_before: Timestamp to retrieve tasks before this
              timestamp.
        """
        params, param_types = _get_params_and_param_types(project_id=project_id,
            config_id=config_id, run_id=_FIRST_JOB_RUN_ID,
            num_tasks=_NUM_OF_TASKS, failure_type=failure_type,
            last_modified_before=last_modified_before)

        query = (("SELECT * FROM %s@{FORCE_INDEX=%s} WHERE %s = @project_id "
                  "AND %s = @config_id AND %s = @run_id "
                  "AND %s = @failure_type ") %
                (SpannerWrapper.TASKS_TABLE,
                 SpannerWrapper.TASK_BY_STATUS_INDEX_NAME,
                 SpannerWrapper.PROJECT_ID,
                 SpannerWrapper.JOB_CONFIG_ID,
                 SpannerWrapper.JOB_RUN_ID,
                 SpannerWrapper.FAILURE_TYPE))

        if last_modified_before is not None:
            query += ("AND %s < @last_modified_before " %
                      SpannerWrapper.LAST_MODIFICATION_TIME)
        query += ("ORDER BY %s DESC LIMIT @num_tasks"
            % SpannerWrapper.LAST_MODIFICATION_TIME)
        return self.list_query(query, params, param_types)

    def get_job_run(self, project_id, config_id, run_id=_FIRST_JOB_RUN_ID):
        """Retrieves the job run with the specified job run id.

        Args:
          project_id: The project id of the desired job run.
          config_id: The config id of the desired job run.
          run_id: The job run id of the desired job run.

        Returns:
          A dictionary containing the job run and job config info.
        """
        query = ('SELECT * FROM {0} JOIN {1} ON {0}.{2} = @config_id AND '
                 '{0}.{3} = @run_id AND {1}.{2} = @config_id').format(
                 SpannerWrapper.JOB_RUNS_TABLE,
                 SpannerWrapper.JOB_CONFIGS_TABLE, SpannerWrapper.JOB_CONFIG_ID,
                 SpannerWrapper.JOB_RUN_ID)
        query = """
            SELECT * FROM %(runs_table)s JOIN %(configs_table)s
            ON %(runs_table)s.%(project_id_col)s = @project_id AND
                %(runs_table)s.%(config_id_col)s = @config_id AND
                %(runs_table)s.%(run_id_col)s = @run_id AND
                %(configs_table)s.%(project_id_col)s = @project_id AND
                %(configs_table)s.%(config_id_col)s = @config_id
            """ % {
                'runs_table': SpannerWrapper.JOB_RUNS_TABLE,
                'configs_table': SpannerWrapper.JOB_CONFIGS_TABLE,
                'project_id_col': SpannerWrapper.PROJECT_ID,
                'config_id_col': SpannerWrapper.JOB_CONFIG_ID,
                'run_id_col': SpannerWrapper.JOB_RUN_ID
            }
        params, param_types = _get_params_and_param_types(project_id=project_id,
            config_id=config_id, run_id=run_id)
        job_run = self.single_result_query(query, params, param_types)
        return job_run

    def delete_job_configs(self, project_id, config_id_list):
        """Deletes a list of job configurations. Only deletes a job
           configuration if it doesn't have tasks in progress.

        Args:
          project_id: The project id of the configs to delete.
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
                project_id, config_id_list, transaction_read_result)
        except ValueError:
            pass # There was nothing to commit. Do nothing.
        return transaction_read_result

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
            if field.name in SpannerWrapper.JSON_FIELDS:
                obj[field.name] = json.loads(result[i])
            else:
                obj[field.name] = result[i]
            i += 1

        return obj
