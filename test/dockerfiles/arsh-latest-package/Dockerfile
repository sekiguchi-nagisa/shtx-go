FROM opensuse/tumbleweed

COPY . /home/tux/dev

RUN zypper addrepo --no-gpgcheck -f https://download.opensuse.org/repositories/home:nsekiguchi/openSUSE_Tumbleweed/home:nsekiguchi.repo && \
    zypper refresh && \
    zypper install -y arsh git go diffutils

WORKDIR /home/tux/dev/
CMD ls ./ && git config --global --add safe.directory "${PWD}" && ./scripts/build.sh && \
    arsh ./test/run_all_tests.ds /usr/share/arsh/tools/litecheck
