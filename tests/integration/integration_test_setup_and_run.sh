# Check for docker install.
docker -v
if [ $? -ne 0 ]
then
  # Install docker if necessary
  sudo curl -sSL https://get.docker.com/ | sh
fi

# Download and extract the worker agent
curl https://storage.googleapis.com/cloud-ingest-pub/gsutil_4.27pre_ing5.tar.gz > ~/gsutil.tar.gz
tar -xvf ~/gsutil.tar.gz

sudo pip --version
if [ $? -ne 0 ]
then
  # Install Python packaging and environment tools
  curl https://bootstrap.pypa.io/get-pip.py > ~/get-pip.py
  sudo python get-pip.py
fi

sudo pip install --upgrade virtualenv
virtualenv ~/venv
sudo ~/venv/bin/pip install --upgrade google-compute-engine
sudo ~/venv/bin/pip install --upgrade google-cloud-storage
sudo ~/venv/bin/pip install --upgrade google-cloud-bigquery

# Download and run the integration test
~/venv/bin/python ~/gsutil/gsutil.py cp gs://cloud-ingest-pub/integration_test.py ~/integration_test.py
~/venv/bin/python ~/integration_test.py
