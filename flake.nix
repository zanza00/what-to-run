{
  description = "A terminal picker for the tasks a project declares";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs =
    { self, nixpkgs }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "aarch64-darwin"
      ];
      eachSystem = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});

      # The release workflow injects the tag into the binaries. A flake build
      # has no tag, so this value keeps the version in one place for Nix users
      # and is bumped with each release.
      version = "0.9.0";

      # go.mod requires a Go version that the default package may not have yet,
      # so prefer the matching attribute when nixpkgs provides it.
      withGo = pkgs: pkgs.buildGoModule.override { go = pkgs.go_1_27 or pkgs.go; };
    in
    {
      packages = eachSystem (
        pkgs:
        let
          buildGoModule = withGo pkgs;
        in
        rec {
          what-to-run = buildGoModule {
            pname = "what-to-run";
            inherit version;

            src = self;

            subPackages = [ "cmd/wtr" ];
            vendorHash = "sha256-0LEYXDfqdXa7597vnwJxalBkRdhsXexuqptvxPZYqt8=";

            # buildGoModule tests only the subPackages. Run the whole module,
            # so a Nix build checks the same packages as `go test ./...`.
            checkPhase = ''
              runHook preCheck
              export GOFLAGS="''${GOFLAGS//-trimpath/}"
              go test ./...
              runHook postCheck
            '';

            env.CGO_ENABLED = 0;
            ldflags = [
              "-s"
              "-w"
              "-X main.version=${version}"
            ];

            meta = {
              description = "A terminal picker for the tasks a project declares";
              homepage = "https://github.com/zanza00/what-to-run";
              license = pkgs.lib.licenses.mit;
              mainProgram = "wtr";
            };
          };

          default = what-to-run;
        }
      );

      apps = eachSystem (pkgs: {
        default = {
          type = "app";
          program = "${self.packages.${pkgs.stdenv.hostPlatform.system}.default}/bin/wtr";
        };
      });

      devShells = eachSystem (pkgs: {
        default = pkgs.mkShell {
          packages = [
            (pkgs.go_1_27 or pkgs.go)
            pkgs.go-task
          ];
        };
      });
    };
}
