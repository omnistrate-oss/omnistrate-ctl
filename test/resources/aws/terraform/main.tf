provider "aws" {
  region = "{{ $sys.deploymentCell.region }}"
}

# Create a Security Group for RDS
resource "aws_security_group" "rds_sg" {
  name        = "e2e-rds-security-group-{{ $sys.id }}"
  description = "Security group for RDS instances"
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

# Create a DB Subnet Group for RDS
resource "aws_db_subnet_group" "rds_subnet_group" {
  name        = "e2e-rds-subnet-group-{{ $sys.id }}"
  description = "My RDS subnet group"

  subnet_ids = [
    "{{ $sys.deploymentCell.publicSubnetIDs[0].id }}",
    "{{ $sys.deploymentCell.publicSubnetIDs[1].id }}",
    "{{ $sys.deploymentCell.publicSubnetIDs[2].id }}"
  ]
}

# Create RDS instances
resource "aws_db_instance" "example1" {
  identifier              = "e2e-instance-1-{{ $sys.id }}"
  engine                  = "mysql"
  instance_class          = "db.t3.micro"
  allocated_storage       = 20
  db_subnet_group_name    = aws_db_subnet_group.rds_subnet_group.name
  vpc_security_group_ids  = [aws_security_group.rds_sg.id]
  username                = "admin"
  password                = "yourpassword"  # Manage securely
  parameter_group_name    = "default.mysql8.0"
  engine_version          = "8.0.37"
  skip_final_snapshot     = true

  depends_on = [
    aws_security_group.rds_sg,
    aws_db_subnet_group.rds_subnet_group
  ]
}

resource "aws_db_instance" "example2" {
  identifier              = "e2e-instance-2-{{ $sys.id }}"
  engine                  = "mysql"
  instance_class          = "db.t3.micro"
  allocated_storage       = 20
  db_subnet_group_name    = aws_db_subnet_group.rds_subnet_group.name
  vpc_security_group_ids  = [aws_security_group.rds_sg.id]
  username                = "admin"
  password                = "yourpassword"
  parameter_group_name    = "default.mysql8.0"
  engine_version          = "8.0.37"
  skip_final_snapshot     = true

  depends_on = [aws_db_instance.example1]
}

output "db_endpoints_1" {
  value = aws_db_instance.example1.endpoint
}

output "db_endpoints_2" {
  value = {
    endpoint: aws_db_instance.example2.endpoint
  }
  sensitive = true
}
