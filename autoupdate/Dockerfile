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

FROM google/cloud-sdk:slim

# Install dependencies.
RUN pip install absl-py

# Copy the autoupdate script.
WORKDIR /cloud-ingest
RUN ["chmod", "757", "/cloud-ingest"]
ADD ./autoupdate.py /cloud-ingest
RUN ["chmod", "755", "/cloud-ingest/autoupdate.py"]
ADD ./THIRD_PARTY_NOTICES /cloud-ingest/THIRD_PARTY_NOTICES

ENTRYPOINT ["./autoupdate.py"]

