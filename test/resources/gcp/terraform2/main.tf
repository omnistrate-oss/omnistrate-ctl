terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "5.34.0"
    }
  }
}

provider "google" {
  project = "{{ $sys.deployment.cloudProviderAccountID }}"
  region = "{{ $sys.deploymentCell.region }}"
}

# Create a pub/sub topic
resource "google_pubsub_topic" "pubsub_app_topic" {
  name = "omnitest-pubsub-topic-{{ $sys.id }}"
}

output "pubsub_id" {
  value = google_pubsub_topic.pubsub_app_topic.id
}

output "redis_endpoint" {
  value = ""
}