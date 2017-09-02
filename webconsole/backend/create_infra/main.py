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

"""Entry point to create infrastructure for cloud ingest."""

import argparse
import os
import time

import cloud_functions_builder
import compute_builder
import job_utilities
import pubsub_builder
import spanner_builder

# pylint: disable=import-error,no-name-in-module
import google.auth as googleauth


DIR = os.path.dirname(os.path.realpath(__file__))


def _parse_topic_subscriptions(topic_subs_name):
    """Parse pubsub command line argument.

    Args:
        topic_subs_name: comma-separated string contains the topic name followed
            by subscription names to assign to this topic.

    Returns:
        Tuple of the (topic name, list of subscription names).

    Raises:
        Exception: if errors encountered.
    """
    topic_subs_arr = topic_subs_name.split(',')
    if len(topic_subs_arr) < 2:
        raise Exception('Error parsing topic/subscriptions from {}. Expecting '
                        'comma-separated string with topic followed by its '
                        'subscriptions.')
    return topic_subs_arr[0], topic_subs_arr[1:]


def tear_down_infrastructure(
    args, spanner_bldr, pubsub_bldr, fn_bldr, compute_bldr):
    """Tears down cloud ingest infrastructure."""
    print 'Deleting GCE instance {}.'.format(args.compute_name)
    compute_bldr.delete_instance(args.compute_name)

    print 'Deleting cloud function {}.'.format(args.function_name)
    fn_bldr.delete_function(args.function_name)

    # Deleting topics.
    for topic_subs_name in args.pubsub:
        topic_name, _ = _parse_topic_subscriptions(topic_subs_name)
        print 'Deleting topic {}, and its subscriptions.'.format(topic_name)
        pubsub_bldr.delete_topic_and_subscriptions(topic_name)

    print 'Deleting spanner instance {}.'.format(args.spanner_instance)
    spanner_bldr.delete_instance()


def create_infrastructure(
    args, spanner_bldr, pubsub_bldr, fn_bldr, compute_bldr):
    """Creates cloud ingest infrastructure."""
    print 'Creating spanner instance {}.'.format(args.spanner_instance)
    spanner_bldr.create_instance()
    print 'Creating database {}.'.format(args.database)
    spanner_bldr.create_database(args.database, os.path.join(DIR, 'schema.ddl'))

    # Creating the PubSub topics/channels.
    for topic_subs_name in args.pubsub:
        topic_name, sub_names = _parse_topic_subscriptions(topic_subs_name)

        print 'Creating topic {}, and subscription {}.'.format(
            topic_name, ','.join(sub_names))
        pubsub_bldr.create_topic_and_subscriptions(topic_name, sub_names)

    print 'Creating cloud function {}.'.format(args.function_name)
    fn_bldr.create_function(args.function_name, args.function_src_dir,
                            args.function_topic, args.function_entrypoint,
                            args.cloud_function_timeout)

    if args.skip_running_dcp:
        print('Skipping create GCE VM for running DCP. All compute arguments '
              'will be ignored')
    else:
        print 'Creating GCE instance {}.'.format(args.compute_name)
        compute_bldr.create_instance(args.compute_name,
                                     args.compute_container_image,
                                     args.compute_cmd,
                                     args.compute_args)


def main():
    """Parses the input args and creates/tears-down infrastructure."""
    _, project_id = googleauth.default()
    parser = argparse.ArgumentParser(
        description='Create infra-structure for cloud ingest')

    parser.add_argument('--spanner-instance', '-s', type=str,
                        help='Name of spanner instance.',
                        default='cloud-ingest-spanner-instance')

    parser.add_argument('--database', '-d', type=str,
                        help='Name of the database.',
                        default='cloud-ingest-database')

    parser.add_argument('--pubsub', '-p', type=str, nargs='+',
                        help='Comma-separated PubSub topic followed by it\'s '
                             'subscriptions',
                        default=[
                            ('cloud-ingest-list-progress,'
                             'cloud-ingest-list-progress'),
                            ('cloud-ingest-copy-progress,'
                             'cloud-ingest-copy-progress'),
                            ('cloud-ingest-loadbigquery-progress,'
                             'cloud-ingest-loadbigquery-progress'),
                            'cloud-ingest-list,cloud-ingest-list',
                            'cloud-ingest-copy,cloud-ingest-copy',
                            ('cloud-ingest-loadbigquery,'
                             'cloud-ingest-loadbigquery')
                        ])

    parser.add_argument('--function-name', '-f', type=str,
                        help='Cloud Function name.',
                        default='cloud-ingest-gcs_to_bq_importer')

    cloud_function_dir = os.path.realpath(
        os.path.join(DIR, '../cloud-functions/gcs-to-bq-importer'))
    parser.add_argument('--function-src-dir', '-sd', type=str,
                        help='Cloud Function source directory.',
                        default=cloud_function_dir)

    parser.add_argument('--function-staging-bucket', '-b', type=str,
                        help='Cloud Function source code staging bucket.',
                        default=None)

    parser.add_argument('--function-staging-object', '-o', type=str,
                        help='Cloud Function source code staging object.',
                        default=None)

    parser.add_argument('--function-topic', '-t', type=str,
                        help='PubSub topic Cloud Function is listening to.',
                        default='cloud-ingest-loadbigquery')

    parser.add_argument('--function-entrypoint', '-e', type=str,
                        help='Cloud Function entry point.',
                        default='GcsToBq')

    parser.add_argument('--cloud-function-timeout', type=str,
                        help='The Cloud Function execution timeout. It '
                             'determines how long before a cloud function is '
                             'considered timed out. Must be a duration in '
                             'seconds, followed by an s. The default is 9 '
                             'minutes.',
                        default='540s')

    parser.add_argument('--skip-running-dcp', '-sdcp', action='store_true',
                        help='Skip running the DCP in a new VM.',
                        default=False)

    parser.add_argument('--compute-name', '-c', type=str,
                        help='GCE instance name.',
                        default='cloud-ingest-dcp')

    parser.add_argument('--compute-container-image', '-i', type=str,
                        help='Container image deployed to the GCE instance.',
                        default='gcr.io/mbassiouny-test/cloud-ingest:dcp')

    parser.add_argument('--compute-cmd', '-l', type=str,
                        help='Command line to run in the container.',
                        default='/cloud-ingest/dcpmain')

    parser.add_argument('--compute-args', '-a', type=str, nargs='+',
                        help='Command line arguments running in the container.',
                        default=[project_id])

    parser.add_argument('--insert-job', '-j', action='store_true',
                        help='Insert a new job into the system.',
                        default=False)

    parser.add_argument('--src-dir', type=str,
                        help='On-prem source directory.',
                        default=None)

    parser.add_argument('--dst-gcs-bucket', type=str,
                        help='GCS destination bucket.',
                        default=None)

    parser.add_argument('--dst-gcs-dir', type=str,
                        help='GCS destination directory in the bucket.',
                        default='')

    parser.add_argument('--dst-bq-dataset', type=str,
                        help='Big query destination dataset.',
                        default=None)

    parser.add_argument('--job-config-name', type=str,
                        help='Name of the job config for the job being inserted',
                        default='ingest-job-00')
  
    parser.add_argument('--job-run-name', type=str,
                        help='Name of the job run for the job being inserted',
                        default='job-run-00')

    parser.add_argument('--dst-bq-table', type=str,
                        help='Big query destination table in the dataset.',
                        default=None)

    parser.add_argument('--force', action='store_true',
                        help='Forcing tear down cloud ingest infrastructure.',
                        default=False)

    parser.add_argument('mode',
                        choices=['Create', 'TearDown', 'CreateThenTearDown'],
                        help='Whether to create or tear down the '
                             'infrastructure.')

    args = parser.parse_args()

    # Make sure that insert job has the sufficient arguments.
    if args.insert_job:
        if (not args.src_dir or not args.dst_gcs_bucket or
            not args.dst_bq_dataset or not args.dst_bq_table):
            parser.error('--insert-job requires --src-dir, --dst-gcs-bucket, '
                         '--dst-bq-dataset and --dst-bq-table')
        if args.mode == 'TearDown':
            parser.error('Can not insert new jobs in the TearDown mode.')

    spanner_bldr = spanner_builder.SpannerBuilder(args.spanner_instance)
    pubsub_bldr = pubsub_builder.PubSubBuilder()
    fn_bldr = cloud_functions_builder.CloudFunctionsBuilder()
    compute_bldr = compute_builder.ComputeBuilder()

    if 'Create' in args.mode:
        create_infrastructure(
            args, spanner_bldr, pubsub_bldr, fn_bldr, compute_bldr)

    database = spanner_bldr.get_database(args.database)
    if args.insert_job:
        job_utilities.create_job(database, args.src_dir,
                                 args.dst_gcs_bucket, args.dst_gcs_dir,
                                 args.dst_bq_dataset, args.dst_bq_table,
                                 args.job_config_name, args.job_run_name)

    if 'TearDown' in args.mode:
        while (not args.force and
               not job_utilities.jobs_have_completed(database)):
            print('Waiting for jobs to complete before tearing down, run with '
                  '--force argument to force the tear down.')
            time.sleep(1)

        tear_down_infrastructure(
            args, spanner_bldr, pubsub_bldr, fn_bldr, compute_bldr)


if __name__ == '__main__':
    main()
