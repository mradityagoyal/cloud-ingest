# Check for docker install.
docker -v
if [ $? -ne 0 ]
then
  # Install docker if necessary.
  sudo curl -sSL https://get.docker.com/ | sh
fi

# Download and extract the worker agent.
curl https://storage.googleapis.com/cloud-ingest-pub/gsutil_4.29pre_ing1.tar.gz > ~/gsutil.tar.gz
tar -xvf ~/gsutil.tar.gz

sudo pip --version
if [ $? -ne 0 ]
then
  # Install Python packaging and environment tools.
  curl https://bootstrap.pypa.io/get-pip.py > ~/get-pip.py
  sudo python get-pip.py
fi

sudo pip install --upgrade virtualenv
virtualenv ~/venv
sudo ~/venv/bin/pip install --upgrade google-compute-engine
sudo ~/venv/bin/pip install --upgrade google-cloud-storage

# Download the integration test.
curl https://raw.githubusercontent.com/GoogleCloudPlatform/cloud-ingest/master/tests/integration/integration_test.py > ~/integration_test.py
if [ "$1" == "run" ]
then
  ~/venv/bin/python ~/integration_test.py
fi
