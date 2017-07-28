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

"""Runs the back-end of the Ingest Web Console, an application written in Flask.

The application is set-up to be a RESTful API.
"""
import httplib
import logging
from corsdecorator import crossdomain  # To allow requests from the front-end
from flask import Flask
from flask import jsonify
from flask import request
from spannerwrapper import SpannerWrapper


app = Flask(__name__)
app.config.from_pyfile('ingestwebconsole.default_settings')
app.config.from_envvar('INGEST_CONFIG_PATH')

spanner_client = SpannerWrapper(
    app.config['JSON_KEY_PATH'],
    app.config['SPANNER_INSTANCE'],
    app.config['SPANNER_DATABASE']
)


@app.route('/jobconfigs', methods=['GET', 'OPTIONS', 'POST'])
@crossdomain(origin=app.config['CLIENT'], headers=['Content-Type'])
def job_configs():
  """Handle all job config related requests."""
  if request.method == 'GET':
    response = jsonify(spanner_client.get_job_configs())
    return response
  elif request.method == 'POST':
    content = request.json
    if 'JobConfigId' not in content or 'JobSpec' not in content:
      response = {
          'error': 'missing required property',
          'message': ('Missing at least one of the required properties: '
                      '[\'JobConfigId\', \'JobSpec\']')
      }
      return jsonify({}), httplib.BAD_REQUEST

    created = spanner_client.create_job_config(content['JobConfigId'],
                                               content['JobSpec'])
    if created:
      created_config = spanner_client.get_job_config(content['JobConfigId'])
      return jsonify(created_config), httplib.CREATED
    else:
      # TODO(b/64075962) Better error handling
      return jsonify({}), httplib.BAD_REQUEST


@app.route('/jobruns', methods=['GET', 'OPTIONS', 'POST'])
@crossdomain(origin=app.config['CLIENT'], headers=['Content-Type'])
def job_runs():
  """Handle all job run related requests."""
  if request.method == 'GET':
    response = jsonify(spanner_client.get_job_runs())
    return response
  elif request.method == 'POST':
    content = request.json
    if 'JobConfigId' not in content or 'JobRunId' not in content:
      response = {
          'error': 'missing required property',
          'message': ('Missing at least one of the required properties: '
                      '[\'JobConfigId\', \'JobRunId\']')
      }
      return jsonify({}), httplib.BAD_REQUEST

    created = spanner_client.create_job_run(content['JobConfigId'],
                                            content['JobRunId'])
    if created:
      created_job_run = spanner_client.get_job_run(content['JobConfigId'],
                                                   content['JobRunId'])
      return jsonify(created_job_run), httplib.CREATED
    else:
      # TODO(b/64075962) Better error handling
      return jsonify({}), httplib.BAD_REQUEST


@app.errorhandler(500)
def server_error(e):
  logging.error('A request could not be completed due to an error: %s', str(e))
  response = {
      'error': 'Internal Server Error',
      'message': ('An internal server error occurred and '
                  'the request could not be completed')
  }
  return jsonify(response), httplib.INTERNAL_SERVER_ERROR


if __name__ == '__main__':
  # Used when running locally
  app.run(host=app.config['HOST'], port=app.config['PORT'],
          debug=app.config['DEBUG'])
