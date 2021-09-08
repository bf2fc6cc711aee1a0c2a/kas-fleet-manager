import logging

import locust, os, random, time, urllib3
from locust import task, HttpUser, constant_pacing

from common.auth import *
from common.tools import *
from common.handler import *

# disable ssl check on workers
urllib3.disable_warnings()

# if set to 'TRUE' - only get endpoints will be attacked
get_only = os.environ['PERF_TEST_GET_ONLY']

# if set to 'TRUE' - db will be seeded with dinosaurs
populate_db = os.environ['PERF_TEST_PREPOPULATE_DB']

# if set to 'TRUE' - db will be seeded with dinosaurs
cleanup_resources = os.environ['PERF_TEST_CLEANUP']

# if PERF_TEST_PREPOPULATE_DB == 'TRUE' - this number determines the number of seed dinosaurs per locust worker
seed_dinosaurs = int(os.environ['PERF_TEST_PREPOPULATE_DB_DINOSAUR_PER_WORKER'])

# PERF_TEST_DINOSAUR_POST_WAIT_TIME specifies number of seconds to wait before creating another dinosaur_request (default is 1)
dinosaur_post_wait_time = int(os.environ['PERF_TEST_DINOSAUR_POST_WAIT_TIME'])

# number of dinosaurs to create by each locust worker
dinosaurs_to_create = int(os.environ['PERF_TEST_DINOSAURS_PER_WORKER'])

# Runtime in minutes
run_time_string = os.environ['PERF_TEST_RUN_TIME']
run_time_seconds = int(run_time_string[0:len(run_time_string)-1]) * 60

# Wait time (in minutes) before hitting endpoints (doesn't apply to prepopulating DB and creating dinosaurs) 
wait_time_in_minutes_string = os.environ['PERF_TEST_HIT_ENDPOINTS_HOLD_OFF']
wait_time_in_minutes = int(wait_time_in_minutes_string)

# set base url for the endpoints (if set via ENV var)
url_base = '/api/dinosaurs_mgmt/v1'
PERF_TEST_BASE_API_URL = os.getenv('PERF_TEST_BASE_API_URL')
if str(PERF_TEST_BASE_API_URL) != 'None':
  url_base = PERF_TEST_BASE_API_URL

# tracks how many dinosaurs were created already (per locust worker)
dinosaurs_created = 0

# service accounts to be created and used by dinosaur clusters
dinosaur_assoc_svc_accs = []

# If 'True' - dinosaur clusters will be created by the worker
is_dinosaur_creation_enabled = True

# if set to 'TRUE' - only one of the workers will be used to create dinosaur clusters
single_worker_dinosaur_create = os.environ['PERF_TEST_SINGLE_WORKER_DINOSAUR_CREATE']

# boolean flag driving cleanup stage
resources_cleaned_up = False

dinosaurs_persisted = False

# used to control population of the dinosaurs.txt config file
dinosaurs_ready = []

dinosaurs_list = []
service_acc_list = []
current_run_time = 0

start_time = time.monotonic()

remove_res_created_start = 90

cleanup_stage_start = 60

# only running GET requests - no need to run cleanup
if (dinosaurs_to_create == 0 and get_only == 'TRUE') or cleanup_resources != "TRUE":
  remove_res_created_start = 0
  cleanup_stage_start = 0

class QuickstartUser(HttpUser):
  def on_start(self):
    global is_dinosaur_creation_enabled
    time.sleep(random.randint(5, 15)) # to make sure that the api server is started
    if single_worker_dinosaur_create == 'TRUE':
      is_dinosaur_creation_enabled = check_dinosaur_create_enabled()
    get_token(self)
  wait_time = constant_pacing(0.5)
  @task
  def main_task(self):
    global current_run_time
    global populate_db
    current_run_time = time.monotonic() - start_time
    # create and then instantly delete dinosaur clusters to seed the database
    if populate_db == 'TRUE' and get_only != 'TRUE':
      if run_time_seconds - current_run_time > 120:
        prepopulate_db(self)
      else:
        populate_db = 'FALSE'
    else:
      # cleanup before the test completes
      if run_time_seconds - current_run_time < remove_res_created_start:
        cleanup(self)
        # make sure that no dinosaur clusters or service accounts created by this test are removed
        if run_time_seconds - current_run_time < cleanup_stage_start:
          if resources_cleaned_up == False:
            check_leftover_resources(self)
      # hit the remaining endpoints for the majority of the time that this test runs
      # between db seed stage (if 'PERF_TEST_PREPOPULATE_DB' env var set to true) and cleanup (1 minute before the end of the test execution)
      else:
        exercise_endpoints(self, get_only)

# pre-populate db with dinosaur clusters
def prepopulate_db(self):
  global populate_db
  global dinosaurs_created
  dinosaur_id = create_dinosaur_cluster(self)
  if dinosaur_id != '':
    remove_resource(self, dinosaurs_list, '/dinosaurs/[id]', dinosaur_id)
    if dinosaurs_created >= seed_dinosaurs:
      time.sleep(60) # wait for all dinosaurs to be deleted
      populate_db = 'FALSE'
      dinosaurs_created = 0

# create dinosaur cluster
def create_dinosaur_cluster(self):
  global dinosaurs_created
  dinosaur_id = handle_post(self, f'{url_base}/dinosaurs?async=true', dinosaur_json(), '/dinosaurs')
  if dinosaur_id != '':
    dinosaurs_created = dinosaurs_created + 1
    dinosaurs_list.append(dinosaur_id)
    persist_dinosaur_id(dinosaur_id)
  return dinosaur_id

# create_svc_acc_for_dinosaur associated with dinosaur cluster
def create_svc_acc_for_dinosaur(self, dinosaur_id):
  global dinosaur_assoc_svc_accs
  # only create service account if not created already
  if get_svc_account_for_dinosaur(dinosaur_assoc_svc_accs, dinosaur_id) == []:
    svc_acc_json_payload = svc_acc_json(url_base)
    svc_acc_id = ''
    while svc_acc_id == '':
      service_acc_json = handle_post(self, f'{url_base}/service_accounts', svc_acc_json_payload, '/service_accounts', True)
      if service_acc_json != '' and 'id' in service_acc_json:
        svc_acc_id = service_acc_json['id']
        dinosaur_assoc_svc_accs.append({'dinosaur_id': dinosaur_id, 'svc_acc_json': service_acc_json})
        persist_service_account_id(svc_acc_id)
      else:
        time.sleep(random.uniform(0.5, 1)) # back off for ~1s if serviceaccount creation was unsuccessful

# main test execution against API endpoints
#
# if get-only is set to 'TRUE', only GET endpoints will be attacked
# otherwise all public API endpoints (where applicable) will be attacked:
# 
# Stages of the main execution include:
# 1. creating dinosaurs followed by creating serviceaccounts (if specified by PERF_TEST_DINOSAURS_PER_WORKER param) 
#    - 1:1 ratio of dinosaurs and service accounts to be created
# 2. attack GET endpoints:
#    - immediately, if PERF_TEST_HIT_ENDPOINTS_HOLD_OFF=0
#    - x minutes after the testing was started, where x is specified by PERF_TEST_HIT_ENDPOINTS_HOLD_OFF parameter
def exercise_endpoints(self, get_only):
  global dinosaurs_created
  global dinosaurs_to_create
  if len(dinosaurs_list) < dinosaurs_to_create and is_dinosaur_creation_enabled == True:
    dinosaur_id = create_dinosaur_cluster(self)
    if dinosaur_id != '':
      if dinosaur_id == '429':
        dinosaurs_to_create = len(dinosaurs_list)
      else:
        time.sleep(dinosaur_post_wait_time) # sleep after creating dinosaur
        create_svc_acc_for_dinosaur(self, dinosaur_id)
  else:
    if dinosaurs_persisted == False and is_dinosaur_creation_enabled == True:
      wait_for_dinosaurs_ready(self)
    # only hit the endpoints, if wait_time_in_minutes has passed already
    if current_run_time / 60 >= wait_time_in_minutes:
      hit_endpoint(self, get_only)
    elif current_run_time / 60 + 1 < wait_time_in_minutes:
      time.sleep(15) # wait 15 seconds instead of hitting this if/else unnecessarily

# The distribution between the endpoints will be semi-random with the following proportions
# dinosaurs get                                       50%
# dinosaur search                                     35%
# dinosaur get                                        10%
# dinosaur metrics                                     1%
# cloud provider(s) get                             1%
# openapi get                                       1%
# service accounts (get, post, delete, reset pwd)   1%
# dinosaur get metrics                      	        0.5%
def hit_endpoint(self, get_only):
  endpoint_selector = random.randrange(0,99)
  if endpoint_selector < 1:
    service_accounts(self, get_only)
  elif endpoint_selector < 2:
    handle_get(self, f'{url_base}/cloud_providers', '/cloud_providers')
    handle_get(self, f'{url_base}/cloud_providers/aws/regions', '/cloud_providers/aws/regions')
  elif endpoint_selector < 3:
    handle_get(self, f'{url_base}/openapi', '/openapi')
  elif endpoint_selector < 53:
    handle_get(self, f'{url_base}/dinosaurs', '/dinosaurs')
  elif endpoint_selector < 88:
    handle_get(self, f'{url_base}/dinosaurs?search={get_random_search()}', '/dinosaurs?search')
  else:
    dinosaur_id = ''
    if len(dinosaurs_list) == 0:
      org_dinosaurs = handle_get(self, f'{url_base}/dinosaurs', '/dinosaurs', True)
      # get all dinosaurs and if the list is not empty - get random dinosaur_id
      items = get_items_from_json_response(org_dinosaurs)
      if len(items) > 0:
        dinosaur_id = get_random_id(get_ids_from_list(items))
    else:
      dinosaur_id = get_random_id(dinosaurs_list)
    if dinosaur_id != '':
      handle_get(self, f'{url_base}/dinosaurs/{dinosaur_id}', '/dinosaurs/[id]')
      if (random.randrange(0,19) < 1): 
        handle_get(self, f'{url_base}/dinosaurs/{dinosaur_id}/metrics/query', '/dinosaurs/[id]/metrics/query')
        handle_get(self, f'{url_base}/dinosaurs/{dinosaur_id}/metrics/query_range?duration=5&interval=30', '/dinosaurs/[id]/metrics/query_range')

# wait for dinosaurs to be in ready state and persist dinosaur config
def wait_for_dinosaurs_ready(self):
  global dinosaurs_persisted
  global dinosaurs_ready
  for dinosaur in dinosaurs_list:
    if not dinosaur in dinosaurs_ready:
      dinosaur_json = handle_get(self, f'{url_base}/dinosaurs/{dinosaur}', '/dinosaurs/[id]', True)
      if 'status' in dinosaur_json:
      # persist service account and dinosaur details to config files
        if dinosaur_json['status'] == 'ready' and 'bootstrapServerHost' in dinosaur_json:
          svc_acc_json_payload = get_svc_account_for_dinosaur(dinosaur_assoc_svc_accs, dinosaur)
          persist_dinosaur_config(dinosaur_json['bootstrapServerHost'], svc_acc_json_payload['svc_acc_json'])
          dinosaurs_ready.append(dinosaur)
          if len(dinosaurs_ready) == len(dinosaurs_list):
            dinosaurs_persisted = True
      else:
        hit_endpoint(self, True)
        time.sleep(random.uniform(1,2)) # sleep before checking dinosaur status again if not ready
      time.sleep(random.uniform(0.4,1)) # sleep before checking another dinosaur

# perf tests against service account endpoints
def service_accounts(self, get_only):
  handle_get(self, f'{url_base}/service_accounts', '/service_accounts')
  if get_only != 'TRUE':
    if len(service_acc_list) > 0:
      remove_resource(self, service_acc_list, '/service_accounts/[id]')
      handle_get(self, f'{url_base}/service_accounts', '/service_accounts')
    else:
      svc_acc_json_payload = svc_acc_json(url_base)
      svc_acc_id = handle_post(self, f'{url_base}/service_accounts', svc_acc_json_payload, '/service_accounts')
      if svc_acc_id != '':
        svc_acc_json_payload['clientSecret'] = generate_random_svc_acc_secret()
        handle_post(self, f'{url_base}/service_accounts/{svc_acc_id}/reset_credentials', svc_acc_json_payload, '/service_accounts/[id]/reset_credentials')
        service_acc_list.append(svc_acc_id)

# get the list of left over service accounts and dinosaur clusters and delete them
def check_leftover_resources(self):
  time.sleep(random.uniform(1.0, 5.0))
  global resources_cleaned_up, service_acc_list, dinosaurs_list
  left_over_dinosaurs = handle_get(self, f'{url_base}/dinosaurs', '/dinosaurs', True)
  # delete all dinosaurs created by the token used in the performance test
  items = get_items_from_json_response(left_over_dinosaurs)
  if len(items) > 0 and dinosaurs_to_create > 0: # only cleanup if any dinosaurs were created by this test
    dinosaurs_list = get_ids_from_list(items)
    for dinosaur_id in dinosaurs_list:
      remove_resource(self, dinosaurs_list, '/dinosaurs/[id]', dinosaur_id)

  time.sleep(random.uniform(1.0, 5.0))
  left_over_svc_accs = handle_get(self, f'{url_base}/service_accounts', '/service_accounts', True)
  # delete all dinosaurs created by the token used in the performance test
  items = get_items_from_json_response(left_over_svc_accs)
  if len(items) > 0:
    for svc_acc_id in get_ids_from_list(items):
      if created_by_perf_test(svc_acc_id, items) == True:
        service_acc_list.append(svc_acc_id)
        remove_resource(self, service_acc_list, '/service_accounts/[id]', svc_acc_id)
  if (len(dinosaurs_list) == 0 and len(service_acc_list) == 0):
    resources_cleaned_up = True

# cleanup created dinosaur clusters and service accounts 1 minute before the test completion
def cleanup(self):
  remove_resource(self, service_acc_list, '/service_accounts/[id]')
  if dinosaurs_to_create > 0: # only delete dinosaurs, if some were created
    remove_resource(self, dinosaurs_list, '/dinosaurs/[id]')

# delete resource from a list
def remove_resource(self, list, name, resource_id = ""):
  if len(list) > 0:
    if resource_id == "":
      resource_id = get_random_id(list)
    if 'dinosaur' in name:
      url = f'{url_base}/dinosaurs/{resource_id}?async=true'
    elif 'service_accounts' in name:
      url = f'{url_base}/service_accounts/{resource_id}'
    status_code = 500
    retry_attempt = 0
    while status_code > 204 and status_code != 404:
      status_code = handle_delete_by_id(self, url, name)
      retry_attempt = retry_attempt + 1
      if status_code != 204 or status_code != 404:
        time.sleep(retry_attempt * random.uniform(0.05, 0.1))
    safe_delete(list, resource_id)
