terraform {
  required_providers {
    onelogin = {
      source = "spbsoluble/onelogin"
    }
  }
}

provider "onelogin" {
  api_url = "https://api.us.onelogin.com"
  # client_id and client_secret can also be set via
  # ONELOGIN_CLIENT_ID and ONELOGIN_CLIENT_SECRET environment variables
}
