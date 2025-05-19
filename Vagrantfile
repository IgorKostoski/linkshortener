# Vagrantfile
Vagrant.configure("2") do |config|
  # This box defaults to x86_64 (amd64) for the virtualbox provider
  config.vm.box = "almalinux/9"

  # You can optionally specify a version if you want to pin it,
  # but usually letting it pick the latest x86_64 is fine.
  # e.g., config.vm.box_version = "9.3.20231113" (check Vagrant Cloud for latest x86_64 version)

  config.vm.box_check_update = false

  config.vm.network "private_network", ip: "192.168.56.10"
  config.vm.network "forwarded_port", guest: 8080, host: 8081 # App port
  config.vm.hostname = "linkshortener-server"

  config.vm.provider "virtualbox" do |vb|
    vb.memory = "2048" # 2GB RAM
    vb.cpus = "2"
    # No special architecture settings needed here for Intel Mac + x86_64 box
  end

  config.vm.provision "shell", inline: <<-SHELL
    echo "Ensuring python3 is available for Ansible..."
    # For AlmaLinux (rpm-based):
    sudo dnf install -y python3 python3-pip
    echo "Python3 check complete."
  SHELL
end