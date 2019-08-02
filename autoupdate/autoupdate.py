#!/usr/bin/python

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

"""autoupdate automatically updates the local agent binary and starts the agent process.

autoupdate runs inside the docker container and is responsible for the update
process of an agent. The script monitors a GCS object and decides if the local
agent needs to be updated. Also, the script monitors if the agent is still
running inside the container. If not, it downloads the stable version of the
agent binary and starts it inside the container.
"""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import argparse
import json
import os
import socket
import subprocess
import sys
import tarfile
import time
import urllib
from absl import flags
from absl import logging

STABLE_AGENT_BINARY_ADDRESS = 'https://www.googleapis.com/storage/v1/b/cloud-ingest-pub/o/agent%2fcurrent%2fagent-linux_amd64.tar.gz'
AGENT_BINARY_FILE_NAME = 'agent-linux_amd64.tar.gz'
POSTFIX = '?alt=media'
# A separate log directory is created under the directory of agent binary log
# to store the logs from this script.
LOG_FOLDER_NAME = '/autoupdate'
CHECK_INTERVAL_SECONDS = 5 * 60

FLAGS = flags.FLAGS
# Flag used in integration tests to pass a test URL as stable agent binary URL.
flags.DEFINE_string('stable_agent_url', STABLE_AGENT_BINARY_ADDRESS,
                    'URL of stable agent binary')
flags.DEFINE_integer('check_interval_seconds', CHECK_INTERVAL_SECONDS,
                     'Agent version check interval')


def delete_agent_source_file(process):
  if process is None:
    return

  try:
    filename = logging.find_log_dir() + '/agent_source_%d.txt' % process.pid
    if os.path.exists(filename):
      os.remove(filename)
  except OSError as err:
    logging.error('Failed to delete file %s, error: %s', filename, str(err))


def agent_release_version(url):
  """Retrieve agent release version.

  Retrieve metadata of the GCS object and the value of version in custom
  metadata.

  Args:
    url: URL of the GCS object.

  Returns:
    A string that represents the version of the agent binary.

  Raises:
    KeyError: An error occurred if the 'AgentVersion' field does not exist in
              the object's metadata.
  """
  version = ''
  response = None
  try:
    response = urllib.urlopen(url).read()
    version = json.loads(response)['metadata']['AgentVersion']
    logging.info('Agent source URL: %s, agent version: %s', url, version)
    return version
  except IOError as err:
    logging.error('Error occurs when opening url %s, error: %s',
                  url, str(err))
  except (ValueError, KeyError) as err:
    logging.error('Error occurs when decoding the response: %s, URL: %s',
                  str(err), url)


def is_process_alive(process):
  if process is None:
    return False

  # poll returns None if the process is not terminated.
  if process.poll() is None:
    return True
  else:
    return False


def extract_agent_binary():
  try:
    tar = tarfile.open(AGENT_BINARY_FILE_NAME)
    tar.extractall()
    tar.close()
  except tarfile.TarError as err:
    logging.error('Failed to extract agent binary file: %s', str(err))


def download_and_start_agent(process, url, args):
  """Downloads the agent binary and starts it locally.

  Downloads the agent from the given URL and extracts the agent binary. Starts
  the agent locally using the arguments passed into the script.

  Args:
    process: Agent process.
    url: URL of the agent binary source in GCS.
    args: Arguments to be passed to agent start command.

  Returns:
    Process id of the local agent.
  """

  try:
    if is_process_alive(process):
      delete_agent_source_file(process)
      process.terminate()
      process.wait()

    download_url = url + POSTFIX
    urllib.urlretrieve(download_url, AGENT_BINARY_FILE_NAME)
    logging.info('Agent is downloaded successfully')

    extract_agent_binary()

    start_args = ['./agent'] + args
    process = subprocess.Popen(start_args)
    logging.info('PID: %d', process.pid)
    return process
  except IOError as ex:
    logging.error('IOError occurs when download and start agent: %s',
                  str(ex))
  except OSError as ex:
    logging.error('OSError occurs when download and start agent: %s',
                  str(ex))


def update_url(process):
  """Get the agent update source URL.

  If there is no agent running locally, return the stable agent URL. Otherwise,
  read the agent source text file to get the update source URL.

  Args:
    process: Agent process or None if there is no currently running process.

  Returns:
    Agent update source URL string.
  """
  if process is None:
    return FLAGS.stable_agent_url

  filename = logging.find_log_dir() + 'agent_source_%d.txt' % process.pid
  try:
    f = open(filename, 'r')
    return f.read()
  except IOError:
    return FLAGS.stable_agent_url


def check_and_update_agent_if_needed(process, local_version, args):
  """Checks the status of the local agent and starts update process if needed.

  Checks the version of the local agent against the version of the uploaded
  agent. If the two versions do not match with each other, start the update
  process. Also, checks if the agent is started successfully and actively
  running. If not, try to download and start the agent again using the stable
  version of agent binary.

  Args:
    process: Agent process or None if there is no currently running process.
    local_version: Version of the local agent.
    args: Arguments passed to this scipt that need to be passed to agent start
          command.

  Returns:
    Process and version of the local agent (whether it was currently running or
    newly started).
  """
  try:
    update_source = update_url(process)
    latest_prod_version = agent_release_version(update_source)

    if latest_prod_version is not None and latest_prod_version != local_version:
      logging.info('Upload version %s does not match local version %s, '
                   'starting update process...',
                   latest_prod_version, local_version)
      process = download_and_start_agent(process, update_source, args)
    else:
      latest_prod_version = local_version

    if not is_process_alive(process):
      # Agent did not start successfully or the previous running agent is not
      # running anymore, we need to backout to the stable version and restart
      # again.
      logging.info('Agent did not start successfully, update source: %s.',
                   update_source)
      update_source = FLAGS.stable_agent_url
      latest_prod_version = agent_release_version(update_source)
      process = download_and_start_agent(process, update_source, args)

    # The local agent does not need to be updated.
    return process, latest_prod_version
  except Exception as ex:  # pylint: disable=broad-except
    logging.error('Unknown exception occurs in '
                  'check_and_update_agent_if_needed, err: %s', str(ex))
    return process, local_version


def setup_logging():
  """Logging related setup.

  Creates the log directory if it does not exist and sets up the output of
  logging to use absl log file.
  """
  logging.set_verbosity(logging.INFO)

  log_dir = ''
  if FLAGS.log_dir:
    log_dir = FLAGS.log_dir + LOG_FOLDER_NAME
  else:
    log_dir = logging.find_log_dir() + LOG_FOLDER_NAME

  try:
    if not os.path.exists(log_dir):
      os.makedirs(log_dir)
    logging.get_absl_handler().use_absl_log_file('', log_dir)
  except OSError as err:
    logging.error('Failed to create log directory, err %s', str(err))
  except Exception as ex:  # pylint: disable=broad-except
    logging.error('Unknown exception occurs in setup_logging, err: %s', str(ex))


def main():
  parser = argparse.ArgumentParser()
  parser.add_argument('--stable_agent_url')
  parser.add_argument('--check_interval_seconds')
  FLAGS(sys.argv, known_only=True)
  # All arguments passed into the auto-update script are stored in unknown and
  # will be passed into the agent start command later.
  _, unknown = parser.parse_known_args()
  unknown.append('--container-id=%s' % socket.gethostname())
  # Remove all empty strings from the arguments list because subprocess.popen
  # stops reading arguments after empty string.
  args = filter(lambda arg: arg != '', unknown)
  logging.info('Arguments that passed to agent: %s', args)

  setup_logging()
  # Temporarily setting the version to be empty and process to be None, these
  # values will be reset later.
  version = ''
  process = None

  while True:
    process, version = check_and_update_agent_if_needed(
        process, version, args)
    time.sleep(FLAGS.check_interval_seconds)

if __name__ == '__main__':
  main()
