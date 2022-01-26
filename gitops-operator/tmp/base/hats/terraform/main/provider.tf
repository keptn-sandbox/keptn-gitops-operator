provider "google-beta" {
  credentials = file("/keptn/terraform/account.json")
  project     = var.gke_project
  region      = var.gke_region
}

provider "google" {
  credentials = file("/keptn/terraform/account.json")
  project     = var.gke_project
  region      = var.gke_region
}

data "google_client_config" "current" {}
