# Use CentOS 7 as the base image
FROM centos:7

# Install Software Collection (SCL) repository
RUN yum install -y centos-release-scl

# Install GCC 7 and make
RUN yum install -y devtoolset-7-gcc devtoolset-7-gcc-c++ make && \
    echo "source scl_source enable devtoolset-7" >> /etc/bashrc

# Install other dependencies
RUN yum install -y astyle openssl-devel python3-pytest python3-pytest-xdist unzip xsltproc doxygen graphviz python3-yaml valgrind git wget

# Install newer version of cmake from source
RUN source /etc/bashrc && \
    wget https://github.com/Kitware/CMake/releases/download/v3.21.1/cmake-3.21.1.tar.gz && \
    tar -zxvf cmake-3.21.1.tar.gz && \
    cd cmake-3.21.1 && \
    ./bootstrap && \
    make && \
    make install

# Install Ninja from source
RUN wget https://github.com/ninja-build/ninja/releases/download/v1.10.2/ninja-linux.zip && \
    unzip ninja-linux.zip && \
    mv ninja /usr/local/bin/

# Set the working directory
WORKDIR /app

# Copy your source code into the container
COPY . .

# Run your build script
RUN source /etc/bashrc && KEM_HELPER_ALL_ARGS=true ./build.sh