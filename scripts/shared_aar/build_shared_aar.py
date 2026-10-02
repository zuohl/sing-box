#!/usr/bin/env python3
"""
build_shared_aar.py
为 sing-box 编译支持「JNI (VPN 模式) + CLI (Root 模式)」双模共享的 libbox.aar。
对主仓库代码零侵入，构建过程在临时 staging 目录中进行，后续同步上游无冲突。
"""

import argparse
import glob
import os
import pathlib
import platform
import re
import shutil
import stat
import subprocess
import sys
import tempfile
import zipfile

ARCHES = {
    'arm64': ('arm64-v8a', 'aarch64-linux-android'),
    'arm': ('armeabi-v7a', 'armv7a-linux-androideabi'),
    '386': ('x86', 'i686-linux-android'),
    'amd64': ('x86_64', 'x86_64-linux-android'),
}

DEFAULT_TAGS = [
    "with_conntrack",
    "with_quic",
    "with_wireguard",
    "with_utls",
    "with_clash_api",
    "with_ebpf",
    "badlinkname",
    "tfogo_checklinkname0",
]


def run_cmd(command, cwd=None, env=None, log_file=None):
    cmd_str = ' '.join(str(c) for c in command)
    print(f'+ {cmd_str}', flush=True)
    if log_file:
        with open(log_file, 'w', encoding='utf-8') as out:
            res = subprocess.run(command, cwd=cwd, env=env, stdout=out, stderr=subprocess.STDOUT)
        with open(log_file, 'r', encoding='utf-8', errors='replace') as out:
            output = out.read()
        if res.returncode != 0:
            raise RuntimeError(f"Command failed (exit code {res.returncode}):\n{output}")
        return output.strip()
    res = subprocess.run(command, cwd=cwd, env=env, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    if res.returncode != 0:
        raise RuntimeError(f"Command failed (exit code {res.returncode}):\n{res.stdout}")
    return res.stdout.strip()


def resolve_ndk():
    ndk_home = os.environ.get('ANDROID_NDK_HOME') or os.environ.get('ANDROID_NDK_ROOT')
    if ndk_home and os.path.isdir(ndk_home):
        return pathlib.Path(ndk_home)
    android_home = os.environ.get('ANDROID_HOME') or os.environ.get('ANDROID_SDK_ROOT')
    if android_home:
        candidates = sorted(glob.glob(os.path.join(android_home, 'ndk', '*')), reverse=True)
        if candidates and os.path.isdir(candidates[0]):
            return pathlib.Path(candidates[0])
    # Fallback to macOS default location
    default_mac = os.path.expanduser('~/Library/Android/sdk/ndk')
    if os.path.isdir(default_mac):
        candidates = sorted(glob.glob(os.path.join(default_mac, '*')), reverse=True)
        if candidates and os.path.isdir(candidates[0]):
            return pathlib.Path(candidates[0])
    raise RuntimeError("Cannot find Android NDK. Please set ANDROID_NDK_HOME environment variable.")


def resolve_clang(ndk_dir):
    sys_name = platform.system()
    host_map = {'Darwin': 'darwin-x86_64', 'Linux': 'linux-x86_64', 'Windows': 'windows-x86_64'}
    host = host_map.get(sys_name, 'linux-x86_64')
    suffix = '.exe' if sys_name == 'Windows' else ''
    clang = ndk_dir / 'toolchains' / 'llvm' / 'prebuilt' / host / 'bin' / ('clang' + suffix)
    if not clang.is_file():
        # Try finding any clang under llvm/prebuilt
        any_clang = list(ndk_dir.glob('toolchains/llvm/prebuilt/*/bin/clang' + suffix))
        if any_clang and any_clang[0].is_file():
            return any_clang[0]
        raise RuntimeError(f"Cannot find clang in NDK: {clang}")
    return clang


def get_version(repo_dir):
    try:
        ver = run_cmd(['git', 'describe', '--tags', '--always'], cwd=repo_dir)
        return ver.strip().lstrip('v')
    except Exception:
        return '1.15.0'


def prepare_staging_sources(repo_dir, stage_dir):
    """
    在 staging 目录中临时准备带有 CLI 转换的源码，保持主仓库 100% 干净
    """
    source_dir = stage_dir / 'sing-box'
    # 拷贝核心工程，跳过 .git 和大文件
    def ignore_patterns(path, names):
        ignored = set()
        if '.git' in names:
            ignored.add('.git')
        if 'bin' in names:
            ignored.add('bin')
        return ignored

    shutil.copytree(repo_dir, source_dir, ignore=ignore_patterns, symlinks=True)

    # 1. 复制 environment 模块到 common/androidcli
    target_env = source_dir / 'common' / 'androidcli'
    target_env.parent.mkdir(parents=True, exist_ok=True)
    scripts_env = repo_dir / 'scripts' / 'shared_aar' / 'environment'
    shutil.copytree(scripts_env, target_env)

    # 2. 改造 cmd/sing-box/*.go，使其成为库包 androidcli 并导出 Run()
    cmd_dir = source_dir / 'cmd' / 'sing-box'
    env_pkg = 'github.com/sagernet/sing-box/common/androidcli'
    for go_file in cmd_dir.glob('*.go'):
        if go_file.name.endswith('_test.go'):
            continue
        text = go_file.read_text(encoding='utf-8')
        text = text.replace('package main', 'package androidcli', 1)
        if go_file.name == 'main.go':
            text = text.replace('func main()', 'func Run()', 1)
        if 'func init() {' in text:
            text = text.replace('func init() {', 'func init() {\n\tif !environment.IsCLI { return }', 1)
            text = text.replace('package androidcli', f'package androidcli\n\nimport environment "{env_pkg}"', 1)
        go_file.write_text(text, encoding='utf-8')

    # 3. 为所有读取 os.Getenv 的包注入环境变量初始化
    for go_file in source_dir.rglob('*.go'):
        if go_file.name.endswith('_test.go') or go_file.is_relative_to(target_env):
            continue
        text = go_file.read_text(encoding='utf-8')
        if re.search(r'os\.(?:Getenv|LookupEnv)\(', text) and env_pkg not in text:
            text, count = re.subn(r'(?m)^(package \w+)\s*$', rf'\1\n\nimport _ "{env_pkg}"', text, count=1)
            if count == 1:
                go_file.write_text(text, encoding='utf-8')

    return source_dir


def package_aar(baseline_aar, target_aar, artifacts):
    """
    将编译出的 libbox.so 和 libsing-box.so 替换/写入 AAR
    """
    replacements = {}
    for abi, (core, launcher) in artifacts.items():
        replacements[f'jni/{abi}/libbox.so'] = core
        replacements[f'jni/{abi}/libsing-box.so'] = launcher

    with zipfile.ZipFile(baseline_aar, 'r') as original, zipfile.ZipFile(target_aar, 'w', zipfile.ZIP_DEFLATED) as out:
        for entry in original.infolist():
            if entry.filename not in replacements:
                out.writestr(entry, original.read(entry))
        for name, path in replacements.items():
            entry = zipfile.ZipInfo(name)
            entry.compress_type = zipfile.ZIP_DEFLATED
            entry.external_attr = (stat.S_IFREG | 0o755) << 16  # 设置可执行权限
            out.writestr(entry, path.read_bytes())


def build(args):
    repo_dir = pathlib.Path(__file__).resolve().parents[2]
    ndk_dir = resolve_ndk()
    clang = resolve_clang(ndk_dir)
    version = get_version(repo_dir)

    print(f"=== 构建双模共享 libbox.aar ===")
    print(f"仓库: {repo_dir}")
    print(f"版本: {version}")
    print(f"NDK: {ndk_dir}")
    print(f"Clang: {clang}")
    print(f"目标架构: {args.arch}")

    work_dir = pathlib.Path(args.work_dir).resolve() if args.work_dir else pathlib.Path(tempfile.mkdtemp(prefix='libbox-shared-'))
    work_dir.mkdir(parents=True, exist_ok=True)

    try:
        stage_dir = work_dir / 'staging'
        stage_dir.mkdir(parents=True, exist_ok=True)

        source_dir = prepare_staging_sources(repo_dir, stage_dir)

        # 环境变量与构建参数
        env = os.environ.copy()
        env['CGO_ENABLED'] = '1'
        env['CGO_LDFLAGS'] = '-Wl,-z,max-page-size=16384'

        tags = ','.join(args.tags)
        ldflags = f"-s -w -buildid= -X github.com/sagernet/sing-box/constant.Version={version} -X runtime.godebugDefault=multipathtcp=0,tlssha1=1 -checklinkname=0"

        # 第一阶段：gomobile bind 生成 baseline AAR
        baseline_aar = stage_dir / 'baseline.aar'
        target_str = ','.join(f'android/{a}' for a in args.arch)
        gomobile_log = work_dir / 'gomobile.log'

        print("正在执行 gomobile bind 生成基础 AAR 与 gobind 源码...")
        gomobile_out = run_cmd([
            'gomobile', 'bind', '-v', '-work',
            '-target', target_str,
            '-androidapi', str(args.api),
            '-javapkg=io.nekohasekai',
            '-libname=box',
            '-tags=' + tags,
            '-trimpath',
            '-buildvcs=false',
            '-ldflags=' + ldflags,
            '-o', str(baseline_aar),
            './experimental/libbox',
        ], cwd=source_dir, env=env, log_file=gomobile_log)

        matches = re.findall(r'^WORK=(.+)$', gomobile_out, re.MULTILINE)
        if not matches:
            raise RuntimeError("gomobile 未返回工作目录 WORK=<dir>")
        gobind_dir = pathlib.Path(matches[-1].strip()) / 'src' / 'gobind'

        # 第二阶段：逐个架构编译 libbox.so（带 CLI 导出）和 libsing-box.so
        artifacts = {}
        jni_dir = stage_dir / 'jni'
        shared_assets_dir = repo_dir / 'scripts' / 'shared_aar'

        for arch in args.arch:
            abi, triple = ARCHES[arch]
            out_abi_dir = jni_dir / abi
            out_abi_dir.mkdir(parents=True, exist_ok=True)
            target_flag = f'--target={triple}{args.api}'

            print(f"正在编译 {abi} 的双模 libbox.so...")
            arch_build_dir = stage_dir / 'build' / arch
            shutil.copytree(gobind_dir, arch_build_dir)
            shutil.copyfile(shared_assets_dir / 'cli_export.go', arch_build_dir / 'cli_export.go')

            arch_env = env.copy()
            arch_env['GOOS'] = 'android'
            arch_env['GOARCH'] = arch
            if arch == 'arm':
                arch_env['GOARM'] = '7'
            arch_env['CC'] = f'"{clang}" {target_flag}'

            so_path = out_abi_dir / 'libbox.so'
            run_cmd([
                'go', 'build', '-buildmode=c-shared',
                '-trimpath', '-buildvcs=false',
                '-tags=' + tags,
                '-ldflags=' + ldflags,
                '-o', str(so_path),
                '.',
            ], cwd=arch_build_dir, env=arch_env)

            print(f"正在编译 {abi} 的微型启动器 libsing-box.so...")
            launcher_so = out_abi_dir / 'libsing-box.so'
            run_cmd([
                str(clang), target_flag,
                '-O2', '-fPIC', '-fPIE', '-pie',
                '-Wall', '-Wextra', '-Werror',
                '-Wl,-z,max-page-size=16384',
                '-Wl,--build-id=none', '-Wl,-s',
                '-o', str(launcher_so),
                str(shared_assets_dir / 'launcher.c'),
                '-ldl',
            ], cwd=source_dir, env=env)

            artifacts[abi] = (so_path, launcher_so)

        # 第三阶段：组装最终 AAR
        out_aar = pathlib.Path(args.output).resolve()
        out_aar.parent.mkdir(parents=True, exist_ok=True)
        print(f"正在组装 AAR 产物: {out_aar}")
        package_aar(baseline_aar, out_aar, artifacts)
        print(f"=== 构建成功: {out_aar} ({out_aar.stat().st_size / 1024 / 1024:.2f} MB) ===")

    finally:
        if not args.work_dir and work_dir.exists():
            shutil.rmtree(work_dir, ignore_errors=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--arch', nargs='+', choices=list(ARCHES.keys()), default=['arm64'])
    parser.add_argument('--api', type=int, default=24)
    parser.add_argument('--tags', nargs='+', default=DEFAULT_TAGS)
    parser.add_argument('--output', default='libbox.aar')
    parser.add_argument('--work-dir', help='保留编译过程的中间文件和日志')
    args = parser.parse_args()
    build(args)


if __name__ == '__main__':
    main()
