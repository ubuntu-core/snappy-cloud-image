[![Build Status](https://travis-ci.org/ubuntu-core/snappy-cloud-image.svg)](https://travis-ci.org/ubuntu-core/snappy-cloud-image) [![Coverage Status](https://coveralls.io/repos/github/ubuntu-core/snappy-cloud-image/badge.svg?branch=master)](https://coveralls.io/github/ubuntu-core/snappy-cloud-image?branch=master)
# Snappy Cloud Image

This utility manages Snappy images in a Glance endpoint to be used by Snappy CI [1].

# Installation

`snappy-cloud-image` has been developed to work on Ubuntu 14.04 or higher. It can be installed using this ppa:

    sudo apt-get install python-software-properties software-properties-common
    sudo add-apt-repository -y ppa:fgimenez/snappy-cloud-image
    sudo apt-get update && sudo apt-get install snappy-cloud-image

# Requirements

At the moment the binary only works for creating images to be used in an OpenStack environment. You should have the common OpenStack environment variables (`$OS_USERNAME`, `$OS_TENANT_NAME`, `$OS_PASSWORD`, `$OS_AUTH_URL` and `$OS_REGION_NAME`) loaded before executing the binary.

# Getting help

You can take a look at the options of the command with:

    snappy-cloud-image -h

# Actions

Each time you invoke the `snappy-cloud-image` command you should pass an `-action` to it, which can be one of:

## create

This action does several things:

* Determines if there's a new image to be created. For this it checks the source endpoint (at http://system-image.ubuntu.com) and the latest image version at the glance endpoint for a given combination of `-release`, `-channel` and `-arch`.

* If there's a new version available then it will:

  * Create a new raw local image using ubuntu-device-flash.

  * Convert the raw image to QCOW2 format.

  * Upload to glance.

## cleanup

With cleanup you can remove the oldest images in glance for a `-release`, `-channel` and `-arch` triplet, keeping the newest 3.

## purge

This action removes all the images created in glance. Use with care!


[1] https://github.com/ubuntu-core/snappy-jenkins
