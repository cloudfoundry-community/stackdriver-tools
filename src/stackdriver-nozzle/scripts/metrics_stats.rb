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
