FROM opensuse/tumbleweed

RUN zypper addrepo --no-gpgcheck -f https://download.opensuse.org/repositories/home:nsekiguchi/openSUSE_Tumbleweed/home:nsekiguchi.repo && \
    zypper refresh && \
    zypper install -y arsh git go diffutils

#RUN curl -L https://ziglang.org/download/0.14.0/zig-linux-x86_64-0.14.0.tar.xz > zig-linux-x86_64-0.14.0.tar.xz && \
#    tar -xf zig-linux-x86_64-0.14.0.tar.xz && mkdir -p /opt && cp -r zig-linux-x86_64-0.14.0 /opt/
#
#ENV PATH=$PATH:/opt/zig-linux-x86_64-0.14.0

COPY . /home/tux/shtx-go

# under Github Actions, regardless of WORKDIR setting, WORKDIR always indicates GITHUB_WORKSPACE (source code location)
# so, if create directory at WORKDIR, need root privilege
# (https://docs.github.com/en/actions/creating-actions/dockerfile-support-for-github-actions)
WORKDIR /home/tux/shtx-go/

CMD DIR="$(pwd)" && cd  /home/tux/shtx-go && git config --global --add safe.directory "${PWD}" && \
    arsh ./scripts/build_all.arsh && \
    cp shtx* /mnt/ && (cp shtx* "$DIR" || true)
