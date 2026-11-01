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

      # go.mod requires Go 1.27.2 or later. Prefer the versioned attribute, and
      # check the default when nixpkgs no longer carries it.
      goFor =
        pkgs:
        if pkgs ? go_1_27 then
          pkgs.go_1_27
        else if pkgs.lib.versionAtLeast pkgs.go.version "1.27.2" then
          pkgs.go
        else
          throw "nixpkgs provides Go ${pkgs.go.version}, but go.mod requires 1.27.2";

      buildGoModuleFor = pkgs: pkgs.buildGoModule.override { go = goFor pkgs; };

      # The release workflow injects the tag into the binaries. A flake build
      # has no tag, so this value keeps the version in one place for Nix users
      # and is checked against the tag by the release workflow.
      version = "0.9.0";
    in
    {
      packages = eachSystem (
        pkgs:
        let
          buildGoModule = buildGoModuleFor pkgs;
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

      devShells = eachSystem (pkgs: {
        default = pkgs.mkShell {
          packages = [
            (goFor pkgs)
            pkgs.go-task
          ];
        };
      });
    };
}
