package GeeCache

import pb "GeeCache/geecachepb"

type PeerPicker interface {
    PickPeer(key string) (PeerGetter, bool)
}

type PeerGetter interface {
    Get(in *pb.Request, out *pb.Response) error
//	Get(group, key string) ([]byte, error)
}

