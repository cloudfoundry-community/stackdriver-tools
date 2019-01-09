#!/usr/bin/env ruby
#
# Copyright 2019 Google Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

#
# Fetch stackdriver binaries and repackage them for a bosh release
#
# Packages:
#  - stackdriver-agent
#

# Fetch latest version
REPO_HOST = 'repo.stackdriver.com'.freeze
APP_HOST = 'app.stackdriver.com'.freeze
CODENAME = `lsb_release -sc`.strip

def exec!(cmd)
  `#{cmd}`
  if $CHILD_STATUS.exitstatus != 0
    raise "Failed to execute command: #{cmd}\nError code: $?.exitstatus"
  end
end

## Add the stackdriver repo
puts 'Adding stackdriver apt repo'
exec! "sudo curl -s -S -f -o /etc/apt/sources.list.d/stackdriver.list \"https://#{REPO_HOST}/#{CODENAME}.list\""
exec! "curl -s -f https://#{APP_HOST}/RPM-GPG-KEY-stackdriver | sudo apt-key add -"
puts 'Updating apt cache'
exec! 'sudo apt-get -q update'

## Create working directory
`mkdir -p stackdriver-agent`

## Download the apt packages needed for stackdriver
puts 'Downloading packages'
`apt-get install -y --download-only stackdriver-agent -o=dir::cache=./stackdriver-agent -o Debug::NoLocking=1`

# Extract and Repackage
puts 'Repackaging'
## Strip .deb and the codename/arch from the package
def strip_end(str)
  str.split('.')[0..-3].join('.')
end

agent_pkg = File.basename(`ls stackdriver-agent/archives/stackdriver-agent*.deb`.strip)

agent_full_name = strip_end(agent_pkg)

def repackage(src_pkg, full_name)
  exec! "mkdir -p stackdriver-agent/extracted/#{full_name}"
  exec! "dpkg -x stackdriver-agent/archives/#{src_pkg} stackdriver-agent/extracted/#{full_name}"
  exec! "tar cfvz stackdriver-agent/#{full_name}.tgz -C stackdriver-agent/extracted/#{full_name}/opt ."

  puts "created: stackdriver-agent/#{full_name}.tgz"
end

repackage(agent_pkg, agent_full_name)

# Clean up
exec! 'rm -rf stackdriver-agent/extracted'
exec! 'rm -rf stackdriver-agent/archives'
exec! 'rm -rf stackdriver-agent/*.bin' # apt-get collateral
