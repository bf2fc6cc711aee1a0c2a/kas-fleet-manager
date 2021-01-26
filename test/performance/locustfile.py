import locust
from locust import task, HttpUser, constant_pacing
import logging

# disable ssl check on workers
import urllib3
urllib3.disable_warnings()

import subprocess
import os
OFFLINE_TOKEN = os.environ['OCM_OFFLINE_TOKEN']

import jwt

import time

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
  assert len(error) == 0,'There should be no error when logging in to OCM'

class QuickstartUser(HttpUser):
  def on_start(self):
    login_to_ocm()
    get_token(self)

  wait_time = constant_pacing(0.005) # to be adjusted later
  @task
  def kafkas_list(self):
    with self.client.get('/api/managed-services-api/v1/kafkas', verify=False, catch_response=True) as response:
      if response.status_code == 401:
        response.success()
        get_token(self)
    