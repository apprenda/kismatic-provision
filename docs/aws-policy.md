# AWS Policy

The following document describes the IAM policy required for creating
resources in AWS.

## Existing VPC

When provisioning nodes in a pre-existing VPC, the following policy is required
for the provisioner:

```
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "Stmt1503669830000",
            "Effect": "Allow",
            "Action": [
                "ec2:CreateTags",
                "ec2:DescribeInstances",
                "ec2:ModifyInstanceAttribute",
                "ec2:RunInstances",
                "ec2:TerminateInstances"
            ],
            "Resource": [
                "*"
            ]
        }
    ]
}
```

## New VPC

When desired, the provisioner is also capable of creating a new VPC, and configuring
all the related resources that are required for the cluster. In this scenario,
the following policy is required:

```
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "Stmt1503669830000",
            "Effect": "Allow",
            "Action": [
                "ec2:DescribeKeyPairs",
                "ec2:CreateKeyPair",
                "ec2:DescribeVpcs",
                "ec2:CreateVpc",
                "ec2:CreateTags",
                "ec2:DescribeSubnets",
                "ec2:CreateSubnet",
                "ec2:AttachInternetGateway",
                "ec2:CreateInternetGateway",
                "ec2:DescribeInternetGateways",
                "ec2:AssociateRouteTable",
                "ec2:CreateRoute",
                "ec2:DescribeRouteTables",
                "ec2:AuthorizeSecurityGroupIngress",
                "ec2:DescribeSecurityGroups",
                "ec2:DescribeInstances",
                "ec2:ModifyInstanceAttribute",
                "ec2:RunInstances",
                "ec2:TerminateInstances"
            ],
            "Resource": [
                "*"
            ]
        }
    ]
}
```