# flake.nix describes a Nix source repository that provides
# development builds of Lanhc and the fork of the Go compiler
# toolchain that Lanhc maintains. It also provides a development
# environment for working on lanhc, for use with "nix develop".
#
# For more information about this and why this file is useful, see:
# https://wiki.nixos.org/wiki/Flakes
#
# Also look into direnv: https://direnv.net/, this can make it so that you can
# automatically get your environment set up when you change folders into the
# project.
#
# WARNING: currently, the packages provided by this flake are brittle,
# and importing this flake into your own Nix configs is likely to
# leave you with broken builds periodically.
#
# The issue is that building Lanhc binaries uses the buildGoModule
# helper from nixpkgs. This helper demands to know the content hash of
# all of the Go dependencies of this repo, in the form of a Nix SRI
# hash. This hash isn't automatically kept in sync with changes made
# to go.mod yet, and so every time we update go.mod while hacking on
# Lanhc, this flake ends up with a broken build due to hash
# mismatches.
#
# Right now, this flake is intended for use by Lanhc developers,
# who are aware of this mismatch and willing to live with it. At some
# point, we'll add automation to keep the hashes more in sync, at
# which point this caveat should go away.
#
# See https://github.com/lanhc/lanhc/issues/6845 for tracking
# how to fix this mismatch.
{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    systems.url = "github:nix-systems/default";
    # Used by shell.nix as a compat shim.
    flake-compat = {
      url = "github:edolstra/flake-compat";
      flake = false;
    };
  };

  outputs = {
    self,
    nixpkgs,
    systems,
    flake-compat,
  }: let
    goVersion = nixpkgs.lib.fileContents ./go.toolchain.version;
    toolChainRev = nixpkgs.lib.fileContents ./go.toolchain.rev;
    flakeHashes = builtins.fromJSON (builtins.readFile ./flakehashes.json);
    gitHash = flakeHashes.toolchain.sri;
    eachSystem = f:
      nixpkgs.lib.genAttrs (import systems) (system:
        f (import nixpkgs {
          system = system;
          overlays = [
            (final: prev: {
              go_1_26 = prev.go_1_26.overrideAttrs (old: {
                version = goVersion;
                src = prev.fetchFromGitHub {
                  owner = "lanhc";
                  repo = "go";
                  rev = toolChainRev;
                  sha256 = gitHash;
                };
                # The Lanhc Go fork carries a placeholder in
                # src/runtime/debug/mod.go that must be replaced with
                # the actual toolchain git rev at build time. Without
                # this, binaries report an empty lanhc.toolchain.rev
                # and the runtime assertion in
                # assert_ts_toolchain_match.go panics.
                postPatch =
                  (old.postPatch or "")
                  + ''
                    substituteInPlace src/runtime/debug/mod.go \
                      --replace-fail "LANHC_GIT_REV_TO_BE_REPLACED_AT_BUILD_TIME" "${toolChainRev}"
                  '';
              });
            })
          ];
        }));
    lanhcRev = self.rev or "";
  in {
    # lanhc takes a nixpkgs package set, and builds Lanhc from
    # the same commit as this flake. IOW, it provides "lanhc built
    # from HEAD", where HEAD is "whatever commit you imported the
    # flake at".
    #
    # This is currently unfortunately brittle, because we have to
    # specify vendorHash, and that sha changes any time we alter
    # go.mod. We don't want to force a nix dependency on everyone
    # hacking on Lanhc, so this flake is likely to have broken
    # builds periodically until someone comes through and manually
    # fixes them up. I sure wish there was a way to express "please
    # just trust the local go.mod, vendorHash has no benefit here",
    # but alas.
    #
    # So really, this flake is for lanhc devs to dogfood with, if
    # you're an end user you should be prepared for this flake to not
    # build periodically.
    packages = eachSystem (pkgs: rec {
      default = pkgs.buildGo126Module {
        name = "lanhc";
        pname = "lanhc";
        src = ./.;
        vendorHash = flakeHashes.vendor.sri;
        nativeBuildInputs = [pkgs.makeWrapper pkgs.installShellFiles];
        ldflags = ["-X lanhc.com/version.gitCommitStamp=${lanhcRev}"];
        env.CGO_ENABLED = 0;
        subPackages = [
          "cmd/lanhc"
          "cmd/lanhcd"
          "cmd/tsidp"
        ];
        doCheck = false;

        # NOTE: We strip the ${PORT} and $FLAGS because they are unset in the
        # environment and cause issues (specifically the unset PORT). At some
        # point, there should be a NixOS module that allows configuration of these
        # things, but for now, we hardcode the default of port 41641 (taken from
        # ./cmd/lanhcd/lanhcd.defaults).
        postInstall =
          pkgs.lib.optionalString pkgs.stdenv.isLinux ''
            wrapProgram $out/bin/lanhcd --prefix PATH : ${pkgs.lib.makeBinPath [pkgs.iproute2 pkgs.iptables pkgs.getent pkgs.shadow]}
            wrapProgram $out/bin/lanhc --suffix PATH : ${pkgs.lib.makeBinPath [pkgs.procps]}

            sed -i \
              -e "s#/usr/sbin#$out/bin#" \
              -e "/^EnvironmentFile/d" \
              -e 's/''${PORT}/41641/' \
              -e 's/$FLAGS//' \
              ./cmd/lanhcd/lanhcd.service

            install -D -m0444 -t $out/lib/systemd/system ./cmd/lanhcd/lanhcd.service
          ''
          + pkgs.lib.optionalString (pkgs.stdenv.buildPlatform.canExecute pkgs.stdenv.hostPlatform) ''
            installShellCompletion --cmd lanhc \
              --bash <($out/bin/lanhc completion bash) \
              --fish <($out/bin/lanhc completion fish) \
              --zsh <($out/bin/lanhc completion zsh)
          '';
      };
      lanhc = default;
    });

    devShells = eachSystem (pkgs: {
      default = pkgs.mkShell {
        packages = with pkgs; [
          curl
          git
          gopls
          gotools
          graphviz
          perl
          go_1_26
          yarn

          # qemu and e2fsprogs are needed for natlab
          qemu
          e2fsprogs

          # mtools (mcopy) and dtc are needed by the `tsapp-qemu-pi`
          # Makefile target that boots the Lanhc appliance under qemu.
          mtools
          dtc

          # awscli2 is used by gokrazy/build.go to import and register the
          # Lanhc appliance AMI.
          awscli2.out
        ];
      };
    });
  };
}
# nix-direnv cache busting line: sha256-TcGcKC18kDxH/MbCBhy1hIxkTfS0/L2j6ft9FESnWb4=
