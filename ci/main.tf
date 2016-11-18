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
  name     = "${var.pipeline}-stackdriver-tools"
  location = "US"
}
