#!/usr/bin/env ruby

require "open-uri"
require "aws-sdk"

module K8sNodeInfo
  CLUSTER_REGEX = /kubernetes.io\/cluster\/(?<cluster_name>[\w-]+)/

  def name_tag
    [role, cluster_name, instance_id].join("-")
  end

  def cluster_name
    instance.tags.each do |tag|
      match = tag.key.match(CLUSTER_REGEX)
      return match[:cluster_name] if match
    end
    raise "Could not determine cluster name from: #{CLUSTER_REGEX.inspect}"
  end

  def eks_endpoint
    eks_describe_cluster("cluster.masterEndpoint")
  end

  def role
    value_from_tag!("Role")
  end

  def node_ip
    metadata("local-ipv4")
  end

  def metadata(name)
    open("http://169.254.169.254/latest/meta-data/#{name}").read
  end

  def instance_id
    metadata("instance-id")
  end

  def availability_zone
    metadata("placement/availability-zone")
  end

  def region
    availability_zone[0..-2]
  end

  def instance
    @_instance ||= Aws::EC2::Instance.new(instance_id, region: region)
  end

  def eks_describe_cluster(query)
    #TODO: when eks is GA we should use the ruby sdk for this
    `aws eks describe-cluster --profile=empty --region=#{region} --cluster-name=#{cluster_name} --query '#{query}'`
  end

  def value_from_tag!(key)
    value_from_tag(key) || raise("Tag: #{key} not found")
  end

  def value_from_tag(key)
    instance.tags.each do |tag|
      return tag.value if tag.key == key
    end
    nil
  end

  def validate_running_as_root
    return if Process.euid == 0
    fail "This script must be run as root!"
  end

  def needs_updating?(path, data)
    File.read(path) != data
  rescue Errno::ENOENT
    true
  end

  def write_config(path, data)
    FileUtils.mkdir_p(File.dirname(path), mode: 0710)
    return unless needs_updating?(path, data)
    File.open(path, "w", 0640) do |file|
      file.write data
    end
    yield if block_given?
  end
end
