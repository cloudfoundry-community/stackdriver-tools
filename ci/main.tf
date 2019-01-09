variable "projectid" {
  type = "string"
}

variable "pipeline" {
  type = "string"
}

variable "region" {
  type    = "string"
  default = "us-east1"
}

provider "google" {
  project = "${var.projectid}"
  region  = "${var.region}"
}

resource "google_storage_bucket" "pipeline-artifacts" {
  name     = "stackdriver-tools-${var.pipeline}"
  location = "US"
}

resource "google_storage_bucket" "bbl-state" {
  name     = "stackdriver-tools-bbl-state-${var.pipeline}"
  location = "US"
}
