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

# if set to 'TRUE' - db will be seeded with kafkas
populate_db = os.environ['PERF_TEST_PREPOPULATE_DB']

# if set to 'TRUE' - db will be seeded with kafkas
cleanup_resources = os.environ['PERF_TEST_CLEANUP']

# if PERF_TEST_PREPOPULATE_DB == 'TRUE' - this number determines the number of seed kafkas per locust worker
seed_kafkas = int(os.environ['PERF_TEST_PREPOPULATE_DB_KAFKA_PER_WORKER'])

# PERF_TEST_KAFKA_POST_WAIT_TIME specifies number of seconds to wait before creating another kafka_request (default is 1)
kafka_post_wait_time = int(os.environ['PERF_TEST_KAFKA_POST_WAIT_TIME'])

# number of kafkas to create by each locust worker
kafkas_to_create = int(os.environ['PERF_TEST_KAFKAS_PER_WORKER'])

# Runtime in minutes
run_time_string = os.environ['PERF_TEST_RUN_TIME']
run_time_seconds = int(run_time_string[0:len(run_time_string)-1]) * 60

# Wait time (in minutes) before hitting endpoints (doesn't apply to prepopulating DB and creating kafkas) 
wait_time_in_minutes_string = os.environ['PERF_TEST_HIT_ENDPOINTS_HOLD_OFF']
wait_time_in_minutes = int(wait_time_in_minutes_string)

# set base url for the endpoints (if set via ENV var)
url_base = '/api/kafkas_mgmt/v1'
PERF_TEST_BASE_API_URL = os.getenv('PERF_TEST_BASE_API_URL')
if str(PERF_TEST_BASE_API_URL) != 'None':
  url_base = PERF_TEST_BASE_API_URL

# tracks how many kafkas were created already (per locust worker)
kafkas_created = 0

# service accounts to be created and used by kafka clusters
kafka_assoc_svc_accs = []

# If 'True' - kafka clusters will be created by the worker
is_kafka_creation_enabled = True

# if set to 'TRUE' - only one of the workers will be used to create kafka clusters
single_worker_kafka_create = os.environ['PERF_TEST_SINGLE_WORKER_KAFKA_CREATE']

# boolean flag driving cleanup stage
resources_cleaned_up = False

kafkas_persisted = False

# used to control population of the kafkas.txt config file
kafkas_ready = []

kafkas_list = []
service_acc_list = []
current_run_time = 0

start_time = time.monotonic()

remove_res_created_start = 90

cleanup_stage_start = 60

# only running GET requests - no need to run cleanup
if (kafkas_to_create == 0 and get_only == 'TRUE') or cleanup_resources != "TRUE":
  remove_res_created_start = 0
  cleanup_stage_start = 0

class QuickstartUser(HttpUser):
  def on_start(self):
    global is_kafka_creation_enabled
    time.sleep(random.randint(5, 15)) # to make sure that the api server is started
    if single_worker_kafka_create == 'TRUE':
      is_kafka_creation_enabled = check_kafka_create_enabled()
    get_token(self)
  wait_time = constant_pacing(0.5)
  @task
  def main_task(self):
    global current_run_time
    global populate_db
    current_run_time = time.monotonic() - start_time
    # create and then instantly delete kafka clusters to seed the database
    if populate_db == 'TRUE' and get_only != 'TRUE':
      if run_time_seconds - current_run_time > 120:
        prepopulate_db(self)
      else:
        populate_db = 'FALSE'
    else:
      # cleanup before the test completes
      if run_time_seconds - current_run_time < remove_res_created_start:
        cleanup(self)
        # make sure that no kafka clusters or service accounts created by this test are removed
        if run_time_seconds - current_run_time < cleanup_stage_start:
          if resources_cleaned_up == False:
            check_leftover_resources(self)
      # hit the remaining endpoints for the majority of the time that this test runs
      # between db seed stage (if 'PERF_TEST_PREPOPULATE_DB' env var set to true) and cleanup (1 minute before the end of the test execution)
      else:
        exercise_endpoints(self, get_only)

# pre-populate db with kafka clusters
def prepopulate_db(self):
  global populate_db
  global kafkas_created
  kafka_id = create_kafka_cluster(self)
  if kafka_id != '':
    remove_resource(self, kafkas_list, '/kafkas/[id]', kafka_id)
    if kafkas_created >= seed_kafkas:
      time.sleep(60) # wait for all kafkas to be deleted
      populate_db = 'FALSE'
      kafkas_created = 0

# create kafka cluster
def create_kafka_cluster(self):
  global kafkas_created
  kafka_id = handle_post(self, f'{url_base}/kafkas?async=true', kafka_json(), '/kafkas')
  if kafka_id != '':
    kafkas_created = kafkas_created + 1
    kafkas_list.append(kafka_id)
    persist_kafka_id(kafka_id)
  return kafka_id

# create_svc_acc_for_kafka associated with kafka cluster
def create_svc_acc_for_kafka(self, kafka_id):
  global kafka_assoc_svc_accs
  # only create service account if not created already
  if get_svc_account_for_kafka(kafka_assoc_svc_accs, kafka_id) == []:
    svc_acc_json_payload = svc_acc_json(url_base)
    svc_acc_id = ''
    while svc_acc_id == '':
      service_acc_json = handle_post(self, f'{url_base}/service_accounts', svc_acc_json_payload, '/service_accounts', True)
      if service_acc_json != '' and 'id' in service_acc_json:
        svc_acc_id = service_acc_json['id']
        kafka_assoc_svc_accs.append({'kafka_id': kafka_id, 'svc_acc_json': service_acc_json})
        persist_service_account_id(svc_acc_id)
      else:
        time.sleep(random.uniform(0.5, 1)) # back off for ~1s if serviceaccount creation was unsuccessful

# main test execution against API endpoints
#
# if get-only is set to 'TRUE', only GET endpoints will be attacked
# otherwise all public API endpoints (where applicable) will be attacked:
# 
# Stages of the main execution include:
# 1. creating kafkas followed by creating serviceaccounts (if specified by PERF_TEST_KAFKAS_PER_WORKER param) 
#    - 1:1 ratio of kafkas and service accounts to be created
# 2. attack GET endpoints:
#    - immediately, if PERF_TEST_HIT_ENDPOINTS_HOLD_OFF=0
#    - x minutes after the testing was started, where x is specified by PERF_TEST_HIT_ENDPOINTS_HOLD_OFF parameter
def exercise_endpoints(self, get_only):
  global kafkas_created
  global kafkas_to_create
  if len(kafkas_list) < kafkas_to_create and is_kafka_creation_enabled == True:
    kafka_id = create_kafka_cluster(self)
    if kafka_id != '':
      if kafka_id == '429':
        kafkas_to_create = len(kafkas_list)
      else:
        time.sleep(kafka_post_wait_time) # sleep after creating kafka
        create_svc_acc_for_kafka(self, kafka_id)
  else:
    if kafkas_persisted == False and is_kafka_creation_enabled == True:
      wait_for_kafkas_ready(self)
    # only hit the endpoints, if wait_time_in_minutes has passed already
    if current_run_time / 60 >= wait_time_in_minutes:
      hit_endpoint(self, get_only)
    elif current_run_time / 60 + 1 < wait_time_in_minutes:
      time.sleep(15) # wait 15 seconds instead of hitting this if/else unnecessarily

# The distribution between the endpoints will be semi-random with the following proportions
# kafkas get                                       50%
# kafka search                                     35%
# kafka get                                        10%
# kafka metrics                                     1%
# cloud provider(s) get                             1%
# openapi get                                       1%
# service accounts (get, post, delete, reset pwd)   1%
# kafka get metrics                      	        0.5%
def hit_endpoint(self, get_only):
  endpoint_selector = random.randrange(1,99) # set to (0,99) post scale testing
  if endpoint_selector < 1:
    service_accounts(self, get_only)
  elif endpoint_selector < 2:
    handle_get(self, f'{url_base}/cloud_providers', '/cloud_providers')
    handle_get(self, f'{url_base}/cloud_providers/aws/regions', '/cloud_providers/aws/regions')
  elif endpoint_selector < 3:
    handle_get(self, f'{url_base}/openapi', '/openapi')
  elif endpoint_selector < 53:
    handle_get(self, f'{url_base}/kafkas', '/kafkas')
  elif endpoint_selector < 88:
    handle_get(self, f'{url_base}/kafkas?search={get_random_search()}', '/kafkas?search')
  else:
    kafka_id = ''
    if len(kafkas_list) == 0:
      org_kafkas = handle_get(self, f'{url_base}/kafkas', '/kafkas', True)
      # get all kafkas and if the list is not empty - get random kafka_id
      items = get_items_from_json_response(org_kafkas)
      if len(items) > 0:
        kafka_id = get_random_id(get_ids_from_list(items))
    else:
      kafka_id = get_random_id(kafkas_list)
    if kafka_id != '':
      handle_get(self, f'{url_base}/kafkas/{kafka_id}', '/kafkas/[id]')
      if (random.randrange(0,19) < 1): 
        handle_get(self, f'{url_base}/kafkas/{kafka_id}/metrics/query', '/kafkas/[id]/metrics/query')
        handle_get(self, f'{url_base}/kafkas/{kafka_id}/metrics/query_range?duration=5&interval=30', '/kafkas/[id]/metrics/query_range')

# wait for kafkas to be in ready state and persist kafka config
def wait_for_kafkas_ready(self):
  global kafkas_persisted
  global kafkas_ready
  for kafka in kafkas_list:
    if not kafka in kafkas_ready:
      kafka_json = handle_get(self, f'{url_base}/kafkas/{kafka}', '/kafkas/[id]', True)
      if 'status' in kafka_json:
      # persist service account and kafka details to config files
        if kafka_json['status'] == 'ready' and 'bootstrapServerHost' in kafka_json:
          svc_acc_json_payload = get_svc_account_for_kafka(kafka_assoc_svc_accs, kafka)
          persist_kafka_config(kafka_json['bootstrapServerHost'], svc_acc_json_payload['svc_acc_json'])
          kafkas_ready.append(kafka)
          if len(kafkas_ready) == len(kafkas_list):
            kafkas_persisted = True
      else:
        hit_endpoint(self, True)
        time.sleep(random.uniform(1,2)) # sleep before checking kafka status again if not ready
      time.sleep(random.uniform(0.4,1)) # sleep before checking another kafka

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

# get the list of left over service accounts and kafka clusters and delete them
def check_leftover_resources(self):
  time.sleep(random.uniform(1.0, 5.0))
  global resources_cleaned_up, service_acc_list, kafkas_list
  left_over_kafkas = handle_get(self, f'{url_base}/kafkas', '/kafkas', True)
  # delete all kafkas created by the token used in the performance test
  items = get_items_from_json_response(left_over_kafkas)
  if len(items) > 0 and kafkas_to_create > 0: # only cleanup if any kafkas were created by this test
    kafkas_list = get_ids_from_list(items)
    for kafka_id in kafkas_list:
      remove_resource(self, kafkas_list, '/kafkas/[id]', kafka_id)

  time.sleep(random.uniform(1.0, 5.0))
  left_over_svc_accs = handle_get(self, f'{url_base}/service_accounts', '/service_accounts', True)
  # delete all kafkas created by the token used in the performance test
  items = get_items_from_json_response(left_over_svc_accs)
  if len(items) > 0:
    for svc_acc_id in get_ids_from_list(items):
      if created_by_perf_test(svc_acc_id, items) == True:
        service_acc_list.append(svc_acc_id)
        remove_resource(self, service_acc_list, '/service_accounts/[id]', svc_acc_id)
  if (len(kafkas_list) == 0 and len(service_acc_list) == 0):
    resources_cleaned_up = True

# cleanup created kafka clusters and service accounts 1 minute before the test completion
def cleanup(self):
  remove_resource(self, service_acc_list, '/service_accounts/[id]')
  if kafkas_to_create > 0: # only delete kafkas, if some were created
    remove_resource(self, kafkas_list, '/kafkas/[id]')

# delete resource from a list
def remove_resource(self, list, name, resource_id = ""):
  if len(list) > 0:
    if resource_id == "":
      resource_id = get_random_id(list)
    if 'kafka' in name:
      url = f'{url_base}/kafkas/{resource_id}?async=true'
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
