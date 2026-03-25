provider "aws" {
  region = "{{ $sys.deploymentCell.region }}"
}

# Create a Security Group for ElastiCache
resource "aws_security_group" "elasticache_sg" {
  name        = "e2e-elasticache-security-group-{{ $sys.id }}"
  description = "Security group for ElastiCache instances"
  vpc_id      = "{{ $sys.deploymentCell.cloudProviderNetworkID }}"

  ingress {
    from_port   = 3306
    to_port     = 3306
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]  # Adjust for appropriate security
  }

  ingress {
    from_port   = 11211  # Default Memcached port
    to_port     = 11211
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]  # Adjust accordingly
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}


# Create a Subnet Group for ElastiCache
resource "aws_elasticache_subnet_group" "elasticache_subnet_group" {
  name       = "e2e-elasticache-subnet-group-{{ $sys.id }}"
  description = "My ElastiCache subnet group"

  subnet_ids = [
    "{{ $sys.deploymentCell.publicSubnetIDs[0].id }}",
    "{{ $sys.deploymentCell.publicSubnetIDs[1].id }}",
    "{{ $sys.deploymentCell.publicSubnetIDs[2].id }}"
  ]
}

# Create ElastiCache Cluster for Memcached
resource "aws_elasticache_cluster" "example_memcached" {
  cluster_id             = "e2e-memcached-{{ $sys.id }}"
  engine                 = "memcached"
  node_type              = "cache.t3.micro"
  num_cache_nodes        = 2  # Adjust as needed

  subnet_group_name      = aws_elasticache_subnet_group.elasticache_subnet_group.name
  security_group_ids     = [aws_security_group.elasticache_sg.id]
}

output "redis_endpoint" {
  value = aws_elasticache_cluster.example_memcached.cache_nodes[0].address
}

output "pubsub_id" {
  value = "" # TODO: Create SQS?
}