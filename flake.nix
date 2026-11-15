{
  description = "Config-driven TUI framework in Go: one JSON file describes a whole interface";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
    let
      systems = [ "x86_64-linux" "aarch64-linux" ];
      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
    in
    {
      packages = forAllSystems (pkgs: rec {
        ultui = pkgs.buildGoModule {
          pname = "ultui";
          version = "0.1.0-unstable-${self.shortRev or self.dirtyShortRev or "dirty"}";
          src = ./.;
          vendorHash = pkgs.lib.fakeHash;
          subPackages = [ "." ];
          env.CGO_ENABLED = 0;
          ldflags = [ "-s" "-w" ];
          postInstall = ''
            mv $out/bin/ultimate-tui $out/bin/ultui
            install -Dm644 README.md DESIGN.md LLM.md -t $out/share/doc/ultui
            install -Dm644 examples/*.json examples/README.md -t $out/share/doc/ultui/examples
          '';
          meta = {
            description = "Config-driven TUI framework in Go: one JSON file describes a whole interface";
            homepage = "https://github.com/Szmelc-INC/Ultimate-Tui";
            license = pkgs.lib.licenses.mit;
            mainProgram = "ultui";
            platforms = pkgs.lib.platforms.linux;
          };
        };
        default = ultui;
      });

      apps = forAllSystems (pkgs: {
        default = {
          type = "app";
          program = "${self.packages.${pkgs.stdenv.hostPlatform.system}.default}/bin/ultui";
        };
      });

      devShells = forAllSystems (pkgs: {
        default = pkgs.mkShell {
          inputsFrom = [ self.packages.${pkgs.stdenv.hostPlatform.system}.default ];
          packages = with pkgs; [ gopls gotools ];
        };
      });
    };
}
