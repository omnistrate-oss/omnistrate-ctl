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
  region  = "{{ $sys.deploymentCell.region }}"
}

# Create a Cloud SQL instance
resource "google_sql_database_instance" "example" {
  name                = "omnitest-sql-{{ $sys.id }}"
  database_version    = "MYSQL_8_0"
  region              = "{{ $sys.deploymentCell.region }}"
  deletion_protection = false

  settings {
    tier = "db-f1-micro"
    ip_configuration {
      ipv4_enabled = true
      authorized_networks {
        name  = "all-networks"
        value = "0.0.0.0/0"
      }
    }
  }
}


resource "google_sql_database" "example_db" {
  name     = "omnitest-db"
  instance = google_sql_database_instance.example.name
}

output "db_endpoints_1" {
  value = google_sql_database_instance.example.connection_name
}

output "db_endpoints_2" {
  value = {
    endpoint : google_sql_database_instance.example.public_ip_address
  }
  sensitive = true
}
