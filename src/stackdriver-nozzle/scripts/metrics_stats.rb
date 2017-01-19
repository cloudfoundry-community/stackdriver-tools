#
# Copyright 2017 Google Inc.
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
#go run main.go | ruby scripts/metrics_stats.rb

require "json"

count = 0
requests = 0
errors = 0

ARGF.each_line.each do |line|
  data = JSON.parse(line)["data"]
  data["counters"] ||= {}

  count += data["counters"].fetch("metrics.count", 0)
  requests += data["counters"].fetch("metrics.requests", 0)
  errors += data["counters"].fetch("metrics.errors", 0)

  puts "Average batch size: #{count / requests.to_f}, errors/request: #{errors / requests.to_f}"
end