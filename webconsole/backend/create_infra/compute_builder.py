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

"""Google Compute admin utilities."""

import copy
import httplib
import json
import time

import googleapiclient.discovery
import google.auth as googleauth


# Cloud config used to initialize the GCE vm and run K8 on it. Running K8
# ensures that the container image is always running and re-spawns it if it's
# died.
_CLOUD_CONFIG = """
#cloud-config
runcmd:
- [
      '/usr/bin/kubelet',
      '--allow-privileged=false',
      '--manifest-url=http://metadata.google.internal/computeMetadata/v1/instance/attributes/google-container-manifest',
      '--manifest-url-header=Metadata-Flavor:Google'
  ]
"""

# GCE instance disk size in GB. 50GB is sufficient for instances running DCP
# container image.
DISK_SIZE_GB = 50


def _wait_operation_to_complete(compute, project_id, zone, operation,
                                timeout_seconds=180):
    """Wait for compute operation to complete."""
    print 'Waiting for operation %s to finish...' % operation
    start_time = time.time()
    while time.time() - start_time < timeout_seconds:
        result = compute.zoneOperations().get(
            project=project_id, zone=zone, operation=operation).execute()

        if result['status'] == 'DONE':
            print 'Operation %s done.' % operation
            if 'error' in result:
                raise Exception(result['error'])
            print 'Operation %s completed in %d seconds.' % (
                operation, time.time() - start_time)
            return result

        time.sleep(1)
    raise Exception(
        'Operation {} timed out.'.format(operation))


class ComputeBuilder(object):
    """Manipulates creation/deletion of GCE instances.

    ComputeBuilder is used to create/delete GCE instances that are initially
    intended to host the cloud ingest data control plane (DCP). Currently, it
    creates GCE instances based on the "Container-Optimized OS from Google"
    image, and runs the cloud ingest DCP container into this instance. Later,
    this can be a generic class for all manipulation of GCE instances.
    """

    def __init__(self, credentials=None, project_id=None):
        self.compute = googleapiclient.discovery.build('compute', 'v1',
                                                       credentials=credentials)
        self.project_id = project_id
        if not self.project_id:
            _, self.project_id = googleauth.default()

        # Getting the optimized container GCE os. This may change in the future
        # if we decide to create generic instance. List of images can be found
        # at https://cloud.google.com/compute/docs/images
        image_response = self.compute.images().getFromFamily(
            project='cos-cloud', family='cos-stable').execute()
        source_disk_image = image_response['selfLink']

        self.container_spec_template = """
        {
            "apiVersion": "v1",
            "kind": "Pod",
            "metadata": {
                "name": "%s"
            },
            "spec": {
                "containers": [
                  {
                      "name": "%s",
                      "image": "%s",
                      "imagePullPolicy": "Always",
                      "command": ["%s"],
                      "args": %s
                  }
                ]
            }
        }
        """

        self.config_template = {
            # 'name': <gce_instance_name>,
            # 'machineType': <machine_type>,

            # Specify the boot disk and the image to use as a source.
            'disks': [
                {
                    'boot': True,  # This is the boot disk
                    'autoDelete': True,  # Auto-delete disk on instance deletion
                    'initializeParams': {
                        'sourceImage': source_disk_image,
                        'diskSizeGb': DISK_SIZE_GB
                    }
                }
            ],

            # Specify a network interface with NAT to access public internet.
            'networkInterfaces': [{
                'network': 'global/networks/default',  # Use the default network
                'accessConfigs': [
                    # Provide external public internet access.
                    {'type': 'ONE_TO_ONE_NAT', 'name': 'External NAT'}
                ]
            }],

            # Allow the instance to access all services.
            'serviceAccounts': [
                {
                    'email': 'default',
                    'scopes': ['https://www.googleapis.com/auth/cloud-platform']
                }
            ],

            # Metadata is readable from the instance and allows you to
            # pass configuration from deployment scripts to instances.
            'metadata': {
                'items': [
                    # Specify the the container running on this instance.
                    # {
                    #     'key': 'google-container-manifest',
                    #     'value': <container_spec>
                    # },
                    {
                        # Initialize K8 on the boot using cloud-config.
                        'key': 'user-data',
                        'value': _CLOUD_CONFIG
                    },
                    {
                        # Ensure containers are always running.
                        'key': 'gci-ensure-gke-docker',
                        'value': 'true'
                    }
                ]
            }
        }

    # pylint: disable=too-many-arguments,too-many-locals
    def create_instance(self, name, container_image, cmd, cmd_args,
                        zone='us-central1-f', machine_type='n1-standard-1'):
        """Creates a GCE instance running a container image.

        Args:
            name: Name of the GCE instance to create.
            container_image: Container image to deploy to the instance.
            cmd: Command line to run in the deployed container.
            cmd_args: Array of params to be passed to the command that runs in
                the deployed container.
            zone: Zone of the GCE instance.
            machine_type: The instance machine type,
                https://cloud.google.com/compute/docs/machine-types lists
                possible values of the machine type.
        """
        args_json_str = '[%s]' % (','.join('"%s"' % x for x in cmd_args))
        container_spec = self.container_spec_template % (
            name, name, container_image, cmd, args_json_str)

        config = copy.deepcopy(self.config_template)
        config['name'] = name
        config['machineType'] = 'zones/%s/machineTypes/%s' % (zone,
                                                              machine_type)
        config['metadata']['items'].append({
            'key': 'google-container-manifest',
            'value': container_spec
        })

        operation = self.compute.instances().insert(
            project=self.project_id, zone=zone, body=config).execute()
        _wait_operation_to_complete(
            self.compute, self.project_id, zone, operation['name'])

    # pylint: enable=too-many-arguments,too-many-locals
    def delete_instance(self, name, zone='us-central1-f'):
        """Deletes a GCE instance."""
        try:
            operation = self.compute.instances().delete(
                project=self.project_id, zone=zone, instance=name).execute()
        except googleapiclient.errors.HttpError as err:
            try:
                error_code = json.loads(
                    err.content.decode('utf-8'))['error']['code']
                if error_code == httplib.NOT_FOUND:
                    print 'GCE instance %s does not exist, skipping delete.' % (
                        name)
                    return
            except:  # pylint: disable=bare-except
                pass
            raise

        _wait_operation_to_complete(
            self.compute, self.project_id, zone, operation['name'])

    def get_instance_status(self, name, zone='us-central1-f'):
        """Gets GCE instance status.

        Args:
            name: Name of the GCE instance.
            zone: GCE instance zone.

        Returns:
            String. 'NOT_FOUND' or a GCE instance resource status defined at
            https://cloud.google.com/compute/docs/reference/latest/instances
        """
        try:
            res = self.compute.instances().get(
                project=self.project_id, zone=zone, instance=name).execute()
        except googleapiclient.errors.HttpError as err:
            if err.resp.status == httplib.NOT_FOUND:
                return 'NOT_FOUND'
            raise

        if 'status' not in res:
            raise Exception('Unexpected error, instance {} status '
                            'missing, response: {}.',
                            name, res)

        return res['status']
