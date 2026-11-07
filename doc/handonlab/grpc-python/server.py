#!/usr/bin/env python3
"""gRPC Profile server minh họa nhiều kiểu dữ liệu.

Cùng hợp đồng api/profile.proto với server Go, nên Go client và Python client
đều gọi được server này.
"""

import argparse
import threading
import time
from concurrent import futures

import grpc

import profile_pb2
import profile_pb2_grpc


class ProfileServicer(profile_pb2_grpc.ProfileServiceServicer):
    def __init__(self) -> None:
        self._lock = threading.Lock()
        self._profiles: list[profile_pb2.Profile] = []

    def SaveProfile(self, request, context):
        profile = request.profile
        if profile is None or not profile.name:
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, "name is required")

        if profile.updated_at_unix == 0:
            profile.updated_at_unix = int(time.time())

        with self._lock:
            self._profiles.append(profile)

        # In toàn bộ kiểu dữ liệu để thấy protobuf giải mã đúng.
        print(
            f"name={profile.name} age={profile.age} "
            f"updated_at={profile.updated_at_unix} active={profile.active} "
            f"score={profile.score:.1f} tags={list(profile.tags)} "
            f"city={profile.address.city} country={profile.address.country} "
            f"role={profile_pb2.Role.Name(profile.role)} "
            f"avatar={profile.avatar!r} attributes={dict(profile.attributes)}",
            flush=True,
        )

        return profile_pb2.ProfileResponse(
            success=True,
            summary=(
                f"Đã nhận hồ sơ của {profile.name} "
                f"(role={profile_pb2.Role.Name(profile.role)})"
            ),
            saved_profile=profile,
        )

    def ListProfiles(self, request, context):
        # Server-streaming RPC: yield nhiều message trên cùng một lời gọi.
        limit = request.limit
        with self._lock:
            snapshot = list(self._profiles)

        for index, profile in enumerate(snapshot):
            if limit > 0 and index >= limit:
                break
            yield profile


def main() -> None:
    parser = argparse.ArgumentParser(description="gRPC Profile server (Python)")
    parser.add_argument("--addr", default="0.0.0.0:50051")
    args = parser.parse_args()

    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    profile_pb2_grpc.add_ProfileServiceServicer_to_server(
        ProfileServicer(), server
    )
    server.add_insecure_port(args.addr)
    server.start()
    print(f"gRPC Profile server (Python) đang chạy tại {args.addr}", flush=True)
    server.wait_for_termination()


if __name__ == "__main__":
    main()
