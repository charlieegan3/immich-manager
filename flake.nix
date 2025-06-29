{
  description = "immich-manager";
  inputs = {
    nixpkgs = {
      type = "github";
      owner = "NixOS";
      repo = "nixpkgs";
      rev = "4b1164c3215f018c4442463a27689d973cffd750";
    };
    flake-utils.url = "github:numtide/flake-utils";
  };
  outputs =
    { nixpkgs, flake-utils, ... }:
    let
      utils = flake-utils;
    in
    utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs {
          inherit system;
          config = {
            allowUnfree = true;
          };
        };
      in
      {
        formatter = pkgs.nixpkgs-fmt;
        
        packages.default = pkgs.buildGoModule {
          pname = "immich-manager";
          version = "0.1.0";
          vendorHash = "sha256-eKeUhS2puz6ALb+cQKl7+DGvm9Cl+miZAHX0imf9wdg=";
          src = ./.;
          checkPhase = "";
        };
        
        devShell = pkgs.mkShell {
          nativeBuildInputs = with pkgs; [
            gci
            gotools
            gofumpt
            go_1_23
            golangci-lint
            claude-code
          ];
          shellHook = '''';
        };
      }
    );
}
