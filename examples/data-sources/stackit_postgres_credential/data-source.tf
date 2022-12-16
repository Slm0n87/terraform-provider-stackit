resource "stackit_postgres_instance" "example" {
  name       = "example"
  project_id = "example"
}

resource "stackit_postgres_credential" "example" {
  project_id  = "example"
  instance_id = stackit_postgres_instance.example.id
}

data "stackit_postgres_credential" "example" {
  id          = stackit_postgres_credential.example.id
  project_id  = "example"
  instance_id = stackit_postgres_instance.example.id
}
