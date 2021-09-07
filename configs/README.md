# Example configuration files

Look to these examples for an understanding of how to set up a configuration for this tool.

# Design Principles

### machines: section

Defines the actual systems and settings for how to manipulated them.

### \*\_templates
These sections are dictionaries to hold templates to reduce redundancy.
Each key in this dictionary can be used by a machine as it's baseline.

# Table of Contents

## sample.yaml
Fairly simple config as an example.

## esxi_kubespray.yaml, hyperv_kubespray.yaml, libvirt_kubespray.yaml
Example scripts that highlight many of the options supported by this tool by showing how to create a cluster of VM's and inventory file to pass to kubespray for installing kubernetes.
