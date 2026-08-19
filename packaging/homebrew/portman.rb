# Homebrew formula for portman. Install the prebuilt binary (no compile).
#
# This file is a template: the SHA256 values are filled in by
# packaging/homebrew/update-formula.sh after a release is published, then the
# result is copied into your tap repo (e.g. forgebay/homebrew-tap).
#
#   brew install forgebay/tap/portman
#
# Note: on Linux the binary links against system GTK + libayatana-appindicator;
# npm or the release tarball is the smoother path there. Homebrew install is
# primarily intended for macOS.
class Portman < Formula
  desc "Menu-bar app to list listening ports, detect runtimes, and kill processes"
  homepage "https://github.com/forgebay/portman"
  version "0.1.0"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/forgebay/portman/releases/download/v#{version}/portman-darwin-arm64"
      sha256 "__SHA_DARWIN_ARM64__"
    end
    on_intel do
      url "https://github.com/forgebay/portman/releases/download/v#{version}/portman-darwin-amd64"
      sha256 "__SHA_DARWIN_AMD64__"
    end
  end

  on_linux do
    url "https://github.com/forgebay/portman/releases/download/v#{version}/portman-linux-amd64"
    sha256 "__SHA_LINUX_AMD64__"
  end

  def install
    bin.install Dir["portman-*"].first => "portman"
  end

  test do
    assert_predicate bin/"portman", :exist?
  end
end
