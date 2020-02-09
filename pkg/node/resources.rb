require 'net/http'
require 'json'
require 'nokogiri'

FILE = "pkg/node/resources.go"

open(FILE, "w") do |file|

file.puts <<-HERDOC
/*
Copyright 2018 Edward Robinson.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package node
HERDOC

instance_types = {}

uri = URI("https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-eni.partial.html")
doc = Nokogiri::HTML(Net::HTTP.get(uri))
table = doc.css(".table-contents table")
table.css("tr").each do |row|
  type, eni, ip, _ = row.css("td").map { |d| d.text.strip.chomp }
  if type
    instance_types[type] = {
      eni: eni,
      ip: ip,
    }
  end
end

uri = URI('https://pricing.us-east-1.amazonaws.com/offers/v1.0/aws/AmazonEC2/current/us-east-1/index.json')
data = JSON.parse(Net::HTTP.get(uri))
data["products"].each do |k, instance|
  if instance["productFamily"] == "Compute Instance"
    instance_type = instance["attributes"]["instanceType"]
  elsif instance["productFamily"] == "Dedicated Host"
    instance_type = instance["attributes"]["instanceType"] + ".metal"
  end

  if(instance_type = instance_types[instance_type])
    instance_type[:cpu] = instance["attributes"]["vcpu"]
    instance_type[:memory] = (instance["attributes"]["memory"].split(" ").first.gsub(",","").to_f * 1024).to_i
  end
end

instance_types = instance_types.reject { |_, i| i[:cpu].nil? or i[:eni].nil? }

# Manually fixup available ips per ENI due to note from https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-eni.html#AvailableIpPerENI
#
# If f1.16xlarge, g3.16xlarge, h1.16xlarge, i3.16xlarge, and r4.16xlarge
# instances use more than 31 IPv4 or IPv6 addresses per interface, they cannot
# access the instance metadata, VPC DNS, and Time Sync services from the 32nd
# IP address onwards. If access to these services is needed from all IP
# addresses on the interface, we recommend using a maximum of 31 IP addresses
# per interface.


instance_types["f1.16xlarge"][:ip] = 31
instance_types["g3.16xlarge"][:ip] = 31
instance_types["h1.16xlarge"][:ip] = 31
instance_types["i3.16xlarge"][:ip] = 31
instance_types["r4.16xlarge"][:ip] = 31

# Manually fixup memory for some metal instances that are not reported
# correctly by the API
#
instance_types["a1.metal"][:memory] = 32 * 1024
instance_types["i3.metal"][:memory] = 512 * 1024
instance_types["i3en.metal"][:memory] = 768 * 1024
instance_types["r5.metal"][:memory] = 768 * 1024
instance_types["m5.metal"][:memory] = 384 * 1024
instance_types["c5.metal"][:memory] = 192 * 1024
instance_types["r5d.metal"][:memory] = 768 * 1024
instance_types["c5n.metal"][:memory] = 192 * 1024
instance_types["c5d.metal"][:memory] = 192 * 1024
instance_types["m5d.metal"][:memory] = 384 * 1024
instance_types["z1d.metal"][:memory] = 384 * 1024
instance_types["u-6tb1.metal"][:memory] = 6291456
instance_types["u-9tb1.metal"][:memory] = 9437184
instance_types["u-12tb1.metal"][:memory] = 12582912
instance_types["u-18tb1.metal"][:memory] = 18874368
instance_types["u-24tb1.metal"][:memory] = 25165824

file.puts "var InstanceCores = map[string]int{"
instance_types.each do |type, info|
  file.puts %Q["#{type}": #{info[:cpu]},]
end
file.puts "}"
file.puts
file.puts "var InstanceMemory = map[string]int{"
instance_types.each do |type, info|
  file.puts %Q["#{type}": #{info[:memory]},]
end
file.puts "}"
file.puts

file.puts "var InstanceENIsAvailable = map[string]int{"
instance_types.each do |type, info|
  file.puts %Q["#{type}": #{info[:eni]},]
end
file.puts "}"
file.puts

file.puts "var InstanceIPsAvailable = map[string]int{"
instance_types.each do |type, info|
  file.puts %Q["#{type}": #{info[:ip]},]
end
file.puts "}"
file.puts

end

system "go", "fmt", FILE
