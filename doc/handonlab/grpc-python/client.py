#!/usr/bin/env python3
"""gRPC Profile client minh họa nhiều kiểu dữ liệu.

Cùng api/profile.proto với client Go, nên Python client gọi được cả server Go
lẫn server Python.
"""

import argparse
import sys

import grpc

import profile_pb2
import profile_pb2_grpc


def main() -> None:
    parser = argparse.ArgumentParser(description="gRPC Profile client (Python)")
    parser.add_argument("--addr", default="localhost:50051")
    parser.add_argument("--name", default="An")
    args = parser.parse_args()

    # insecure chỉ dùng cho lab chạy local. Production phải dùng TLS.
    with grpc.insecure_channel(args.addr) as channel:
        stub = profile_pb2_grpc.ProfileServiceStub(channel)

        profile = profile_pb2.Profile(
            name=args.name,
            age=21,
            active=True,
            score=9.5,
            tags=["python", "grpc", "protobuf"],
            address=profile_pb2.Address(
                city="Hà Nội",
                country="Việt Nam",
            ),
            role=profile_pb2.ROLE_EDITOR,
            avatar=b"demo-avatar",
            attributes={"team": "backend", "level": "beginner"},
        )

        # 1. Unary RPC: 1 request -> 1 response.
        try:
            reply = stub.SaveProfile(
                profile_pb2.ProfileRequest(profile=profile),
                timeout=5,
            )
        except grpc.RpcError as exc:
            print(f"SaveProfile failed: {exc.code()} {exc.details()}")
            sys.exit(1)

        print("success:", reply.success)
        print("summary:", reply.summary)
        print("saved.name:", reply.saved_profile.name)
        print("saved.updated_at_unix:", reply.saved_profile.updated_at_unix)
        print("saved.attributes:", dict(reply.saved_profile.attributes))

        # 2. Server-streaming RPC: nhận nhiều Profile trên cùng một lời gọi.
        print("\ndanh sách hồ sơ từ server stream:")
        try:
            for item in stub.ListProfiles(
                profile_pb2.ListProfilesRequest(limit=10),
                timeout=5,
            ):
                print(
                    f"- {item.name} | "
                    f"role={profile_pb2.Role.Name(item.role)} | "
                    f"tags={list(item.tags)} | city={item.address.city}"
                )
        except grpc.RpcError as exc:
            print(f"ListProfiles failed: {exc.code()} {exc.details()}")
            sys.exit(1)


if __name__ == "__main__":
    main()
