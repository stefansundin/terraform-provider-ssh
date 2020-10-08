# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure("2") do |config|

  config.vm.box = "debian/buster64"

  config.vm.provision "shell", inline: <<-SHELL
    export DEBIAN_FRONTEND="noninteractive"
    sudo debconf-set-selections <<< "mariadb-server mysql-server/root_password password SECRET"
    sudo debconf-set-selections <<< "mariadb-server mysql-server/root_password_again password SECRET"

    apt-get update
    apt-get install -y mariadb-server python-mysqldb
  SHELL
end
