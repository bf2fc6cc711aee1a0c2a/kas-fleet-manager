import locust
from locust import task, HttpUser, constant_pacing
import logging

# disable ssl check on workers
import urllib3
urllib3.disable_warnings()

import subprocess
import os
OFFLINE_TOKEN = os.environ['OCM_OFFLINE_TOKEN']

# if set to "TRUE" - db will be seeded with kafkas
populate_db = os.environ['PERF_TEST_PREPOPULATE_DB']
# if PERF_TEST_PREPOPULATE_DB == "TRUE" - this number determines the number of seed kafkas per locust user
seed_kafkas = int(os.environ["PERF_TEST_PREPOPULATE_DB_KAFKA_PER_USER"])

import jwt

import time

import random, string

kafkas_created = 0

# get ocm token on each of the workers and set headers to use the token when talking to the API
def get_token(self):
  global short_living_token
  
  logging.info('Obtaining fresh OCM token')

  current_time = time.time()

  # ignored exception will be thrown here (safe to ignore): https://bugs.python.org/issue42350
  get_token_output = subprocess.run(['ocm', 'token'], stdout=subprocess.PIPE)

  short_living_token = str(get_token_output.stdout.decode('utf-8')).rstrip('\n')
  
  assert len(short_living_token) > 0,'OCM short-living token shouldn\'t be empty'

  token_decoded = jwt.decode(short_living_token, options={'verify_signature': False})
  
  token_expiry = token_decoded['exp']

  assert token_expiry > current_time,'Fresh token shouldn\'t be expired'

  self.client.headers = {'Content-Type': 'application/json', 'Authorization': 'Bearer ' + short_living_token}

# login to ocm to be able to retrieve ocm token on each of the workers
def login_to_ocm():
  logging.info('Logging in to ocm in order to be able to retrieve fresh ocm tokens')
    
  output, error = subprocess.Popen(f'ocm login --url=https://api.stage.openshift.com/ --token="{OFFLINE_TOKEN}"' , universal_newlines=True, shell=True,
    stdout=subprocess.PIPE, stderr=subprocess.PIPE).communicate()

  assert len(output) == 0,'There should be no output when logging in to OCM'

def random_string(length):
   letters = string.ascii_lowercase
   return ''.join(random.choice(letters) for i in range(length))

class QuickstartUser(HttpUser):
  def on_start(self):
    login_to_ocm()
    get_token(self)

  wait_time = constant_pacing(0.05) # to be adjusted later
  @task
  def main_task(self):
    global populate_db
    global kafkas_created
    # create and then instantly delete kafka_requests to seed the database
    if populate_db == "TRUE":
      # logging.info(run_time)
      json = {
        "cloud_provider": "aws",
        "region": "us-east-1",
        "name": random_string(15),
        "multi_az": True
      }
      with self.client.post('/api/managed-services-api/v1/kafkas?async=true', json=json, verify=False, catch_response=True) as response:
        if response.status_code == 409:
          response.success() # ignore unlike 409 errors when generated kafka name is duplicated
        if response.status_code == 401:
          response.success()
          get_token(self)
        if response.status_code == 202:
          kafka_id = response.json()["id"]
          kafkas_created = kafkas_created + 1
          with self.client.delete(f'/api/managed-services-api/v1/kafkas/{kafka_id}?async=true', json=json, verify=False, catch_response=True, name="/api/managed-services-api/v1/kafkas/[id]") as response:
            assert response.status_code == 204,'Unexpected status code for kafka deletion. Manual cleanup required!'
            if kafkas_created >= seed_kafkas:
              time.sleep(120) # wait for all kafkas to be deleted
              populate_db = "FALSE"
              wait_time = constant_pacing(0.00005)
    else:
      with self.client.get('/api/managed-services-api/v1/kafkas', verify=False, catch_response=True) as response:
        if response.status_code == 401:
          response.success()
          get_token(self)
    