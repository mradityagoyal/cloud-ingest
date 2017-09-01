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


# TODO(b/63017649): Remove the hard coded JobConfigId and JobRunId.
JOB_CONFIG_NAME = 'ingest-job-00'
JOB_RUN_NAME = 'job-run-00'

TASK_STATUS_UNQUEUED = 0
TASK_STATUS_QUEUED = 1
TASK_STATUS_FAILED = 2
TASK_STATUS_SUCCESS = 3

TASK_TYPE_LIST = 1

def JobsHaveCompleted(database):
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


def CreateJob(database, src_dir, dst_gcs_bucket, dst_gcs_dir,
              dst_bq_dataset, dst_bq_table):
  """Creates a new transfer job into the spanner database."""
  with database.batch() as batch:
    # Adding job config.
    job_spec = {
        'onPremSrcDirectory': src_dir,
        'gcsBucket': dst_gcs_bucket,
        'gcsDirectory': dst_gcs_dir,
        'bigqueryDataset': dst_bq_dataset,
        'bigqueryTable': dst_bq_table
    }
    batch.insert(
        table='JobConfigs',
        columns=('JobConfigId', 'JobSpec'),
        values=[(JOB_CONFIG_NAME, json.dumps(job_spec))])

    # Adding a job run.
    batch.insert(
        table='JobRuns',
        columns=('JobConfigId', 'JobRunId'),
        values=[(JOB_CONFIG_NAME, JOB_RUN_NAME)]
    )

    # Adding the listing task
    task_id = 'list'
    list_result_object_name = os.path.join(
        dst_gcs_dir, 'list-task-output-%s-%s-%s' % (JOB_CONFIG_NAME,
                                                    JOB_RUN_NAME,
                                                    task_id))
    task_spec = {
        'dst_list_result_bucket': dst_gcs_bucket,
        'dst_list_result_object': list_result_object_name,
        'src_directory': src_dir
    }

    timestamp = int(time.time() * 1e9)

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
        values=[(JOB_CONFIG_NAME,
                 JOB_RUN_NAME,
                 task_id,
                 json.dumps(task_spec).encode('utf-8'),
                 TASK_TYPE_LIST,
                 TASK_STATUS_UNQUEUED,
                 timestamp,
                 timestamp)])
