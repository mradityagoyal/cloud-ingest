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

"""Cloud ingest transfer jobs utilities."""

import json
import os
import time

TASK_STATUS_UNQUEUED = 0
TASK_STATUS_QUEUED = 1
TASK_STATUS_FAILED = 2
TASK_STATUS_SUCCESS = 3

TASK_TYPE_LIST = 1

JOB_STATUS_IN_PROGRESS = 1



def jobs_have_completed(database):
    """Check whether all jobs in the systems have completed."""
    if not database.exists():
        return True
    results = database.execute_sql(
        'SELECT COUNT(*) '
        'FROM Tasks@{FORCE_INDEX=TasksByStatus} '
        'WHERE Status != %d AND Status != %d' % (TASK_STATUS_FAILED,
                                                 TASK_STATUS_SUCCESS))
    for row in results:
        print 'There are %d tasks still processing.' % row[0]
        return row[0] == 0

# pylint: disable=too-many-arguments,too-many-locals
def create_job(database, src_dir, dst_gcs_bucket, dst_gcs_dir,
               config_name, run_name):
    """Creates a new transfer job into the spanner database."""
    with database.batch() as batch:
        # Adding job config.
        timestamp = int(time.time() * 1e9)
        job_spec = {
            'onPremSrcDirectory': src_dir,
            'gcsBucket': dst_gcs_bucket,
            'gcsDirectory': dst_gcs_dir
        }
        batch.insert(
            table='JobConfigs',
            columns=('JobConfigId', 'JobSpec'),
            values=[(config_name, json.dumps(job_spec))])

        job_counters = {
            # Overall job run stats.
            'totalTasks': 1,  # Start at 1 b/c list task is manually inserted
            'tasksCompleted': 0,
            'tasksFailed': 0,
            'tasksQueued': 0,
            'tasksUnqueued': 1,

            # List task stats.
            'totalTasksList': 1,
            'tasksCompletedList': 0,
            'tasksFailedList': 0,
            'tasksQueuedList': 0,
            'tasksUnqueuedList': 1,

            # Copy task stats.
            'totalTasksCopy': 0,
            'tasksCompletedCopy': 0,
            'tasksFailedCopy': 0,
            'tasksQueuedCopy': 0,
            'tasksUnqueuedCopy': 0
        }

        # Adding a job run.
        batch.insert(
            table='JobRuns',
            columns=('JobConfigId',
                     'JobRunId',
                     'JobCreationTime',
                     'Status',
                     'Counters'),
            values=[(
                config_name, run_name, timestamp, JOB_STATUS_IN_PROGRESS,
                json.dumps(job_counters)
            )]
        )

        # Adding the listing task
        task_id = 'list'
        list_result_object_name = os.path.join(
            dst_gcs_dir, 'list-task-output-%s-%s-%s' % (config_name,
                                                        run_name,
                                                        task_id))
        task_spec = {
            'dst_list_result_bucket': dst_gcs_bucket,
            'dst_list_result_object': list_result_object_name,
            'src_directory': src_dir
        }

        batch.insert(
            table='Tasks',
            columns=('JobConfigId',
                     'JobRunId',
                     'TaskId',
                     'TaskSpec',
                     'TaskType',
                     'Status',
                     'CreationTime',
                     'LastModificationTime'),
            values=[(config_name,
                     run_name,
                     task_id,
                     json.dumps(task_spec).encode('utf-8'),
                     TASK_TYPE_LIST,
                     TASK_STATUS_UNQUEUED,
                     timestamp,
                     timestamp)])

# pylint: disable=too-many-arguments,too-many-locals
