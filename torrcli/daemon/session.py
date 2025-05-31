import libtorrent as lt
from torrcli.daemon.config import TORRENT_DATA_DIR

ses = lt.session({'listen_interfaces': '0.0.0.0:6881'})
torrent_handles = {}

def load_resume_and_torrents():
    for fastresume_file in TORRENT_DATA_DIR.glob("*.fastresume"):
        info_hash = fastresume_file.stem
        resume_data = fastresume_file.read_bytes()
        torrent_file = TORRENT_DATA_DIR / (info_hash + ".torrent")

        try:
            if torrent_file.exists():
                ti = lt.torrent_info(str(torrent_file))
                atp = lt.read_resume_data(resume_data)
                if not atp.ti:
                    atp.ti = ti

                handle = ses.add_torrent(atp)
                handle.auto_managed(True)
                torrent_handles[info_hash] = handle
                print(f"Loaded torrent {ti.name()} with resume data")
            else:
                print(f"Missing torrent file for info hash {info_hash}, skipping fast resume")
        except Exception as e:
            print(f"Failed to load resume data for {info_hash}: {e}")

def save_all_resume_data():
    for handle in ses.get_torrents():
        handle.save_resume_data()

def on_save_resume_data(alert):
    handle = alert.handle
    info_hash = str(handle.info_hashes().get_best()) 
    data = lt.bencode(alert.resume_data)
    file_path = TORRENT_DATA_DIR / f"{info_hash}.fastresume"
    try:
        file_path.write_bytes(data)
        print(f"Saved fast resume data for {info_hash}")
    except Exception as e:
        print(f"Failed to save fast resume data for {info_hash}: {e}")