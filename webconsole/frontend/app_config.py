"""
This script is run while configuring the angular application. The logic on this
file runs before the application is started with npm start.
"""

import os
import string
import sys
import collections


def exit_api_not_set_error():
    if not INGEST_API_URL:
        print('ERROR: OPI_API_URL variable not set. Please set the OPI_API_URL '
              'environment variable to the API url and run again.')
        sys.exit(1)

def exit_robot_account_not_set_error():
    if not ROBOT_ACCOUNT:
        print('ERROR: OPI_ROBOT_ACCOUNT variable not set. Please set the '
              'OPI_ROBOT_ACCOUNT environment variable.')
        sys.exit(1)

# TODO: Define different API URLS for prod and test environments.
INGEST_API_URL = ''
try:
    INGEST_API_URL = os.environ['OPI_API_URL']
except:
    exit_api_not_set_error()

# TODO: Define different service accounts for prod and test environments.
ROBOT_ACCOUNT = ''
try:
    ROBOT_ACCOUNT = os.environ['OPI_ROBOT_ACCOUNT']
except:
    exit_robot_account_not_set_error()

if not INGEST_API_URL:
    exit_api_not_set_error()
if not ROBOT_ACCOUNT:
    exit_robot_account_not_set_error()

Environment = collections.namedtuple('Environment',
                                     'filename client_id is_prod account '
                                     'pub_sub_prefix')

# The environment files to write.
ENVIRONMENTS = [
    Environment(
        filename='environment.prod.ts',
        client_id=
        '342921335261-7c1jvv8175oaj9m68fgnot2jl786h7on.apps.googleusercontent.com',
        account=ROBOT_ACCOUNT,
        is_prod='true',
        pub_sub_prefix=''),
    Environment(
        filename='environment.ts',
        client_id=
        '701178595865-por9ijjvgbjoka841c1mkki23tqka66a.apps.googleusercontent.com',
        account=ROBOT_ACCOUNT,
        is_prod='false',
        pub_sub_prefix='test-')
]

# The directory with the environment files.
ENV_DIRECTORY = 'src/environments/'

TEMPLATE = string.Template("""
export const environment = {
  production: '$IS_PRODUCTION',
  apiUrl: '$API_URL',
  authClientId: '$AUTH_CLIENT_ID',
  robotAccountEmail: '$ROBOT_ACCOUNT_EMAIL',
  pubSubPrefix: '$PUBSUB_TOPIC_PREFIX',
};
""")

if not os.path.exists(ENV_DIRECTORY):
    os.makedirs(ENV_DIRECTORY)

for environment in ENVIRONMENTS:
    with open(ENV_DIRECTORY + environment.filename, "w") as env_file:
        env_file.write(
            TEMPLATE.substitute(
                IS_PRODUCTION=environment.is_prod,
                API_URL=INGEST_API_URL,
                AUTH_CLIENT_ID=environment.client_id,
                ROBOT_ACCOUNT_EMAIL=environment.account,
                PUBSUB_TOPIC_PREFIX=environment.pub_sub_prefix))
