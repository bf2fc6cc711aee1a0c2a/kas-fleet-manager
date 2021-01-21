import locust
from locust import task, HttpUser, constant_pacing

# disable ssl check on workers
import urllib3
urllib3.disable_warnings()

class QuickstartUser(HttpUser):
  wait_time = constant_pacing(0.0001) # to be adjusted later

  @task
  def kafkas_list(self):
    # self.client.get("/api/managed-services-api/v1/kafkas", verify=False, headers={"Content-Type": "application/json", "Authorization": "Bearer " + SHORT_LIVING_TOKEN})