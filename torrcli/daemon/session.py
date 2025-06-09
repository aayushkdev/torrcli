import libtorrent as lt
from torrcli.daemon.config import (
    DATA_DIR,
    LISTEN_INTERFACES,
    DHT_ENABLED,
    LSD_ENABLED,
    UPNP_ENABLED,
    NATPMP_ENABLED,
    INCOMING_UTP_ENABLED,
    OUTGOING_UTP_ENABLED,
    INCOMING_TCP_ENABLED,
    OUTGOING_TCP_ENABLED,
    MAX_DOWNLOAD_SPEED,
    MAX_UPLOAD_SPEED,
    MAX_ACTIVE_DOWNLOADS,
    MAX_ACTIVE_SEEDS,
    SHARE_RATIO_LIMIT,
    AUTO_START,
    ANONYMOUS_MODE,
    VALIDATE_HTTPS_TRACKERS,
    DHT_PRIVACY_LOOKUPS,
    DHT_IGNORE_DARK_INTERNET,
    SSRF_MITIGATION,
    OUT_ENC_POLICY,
    IN_ENC_POLICY,
    ALLOWED_ENC_LEVEL,
    PREFER_RC4,
    PROXY_TYPE,
    PROXY_HOSTNAME,
    PROXY_PORT,
    PROXY_USERNAME,
    PROXY_PASSWORD,
    PROXY_HOSTNAMES,
    PROXY_PEER_CONNECTIONS,
    PROXY_TRACKER_CONNECTIONS,
    FORCE_PROXY
)


def create_session():
    ses = lt.session()

    ses.set_alert_mask(lt.alert.category_t.status_notification)

    settings = ses.get_settings()

    settings["listen_interfaces"] = LISTEN_INTERFACES
    settings["enable_dht"] = DHT_ENABLED
    settings["enable_lsd"] = LSD_ENABLED
    settings["enable_upnp"] = UPNP_ENABLED
    settings["enable_natpmp"] = NATPMP_ENABLED
    settings["enable_outgoing_utp"] = OUTGOING_UTP_ENABLED
    settings["enable_incoming_utp"] = INCOMING_UTP_ENABLED
    settings["enable_outgoing_tcp"] = OUTGOING_TCP_ENABLED
    settings["enable_incoming_tcp"] = INCOMING_TCP_ENABLED

    settings["active_downloads"] = MAX_ACTIVE_DOWNLOADS
    settings["active_seeds"] = MAX_ACTIVE_SEEDS
    settings["share_ratio_limit"] = SHARE_RATIO_LIMIT

    settings["anonymous_mode"] = ANONYMOUS_MODE
    settings["validate_https_trackers"] = VALIDATE_HTTPS_TRACKERS
    settings["dht_privacy_lookups"] = DHT_PRIVACY_LOOKUPS
    settings["dht_ignore_dark_internet"] = DHT_IGNORE_DARK_INTERNET
    settings["ssrf_mitigation"] = SSRF_MITIGATION
    settings["out_enc_policy"] = OUT_ENC_POLICY
    settings["in_enc_policy"] = IN_ENC_POLICY
    settings["allowed_enc_level"] = ALLOWED_ENC_LEVEL
    settings["prefer_rc4"] = PREFER_RC4

    settings["proxy_type"] = PROXY_TYPE
    settings["proxy_hostname"] = PROXY_HOSTNAME
    settings["proxy_port"] = PROXY_PORT
    settings["proxy_username"] = PROXY_USERNAME
    settings["proxy_password"] = PROXY_PASSWORD
    settings["proxy_hostnames"] = PROXY_HOSTNAMES
    settings["proxy_peer_connections"] = PROXY_PEER_CONNECTIONS
    settings["proxy_tracker_connections"] = PROXY_TRACKER_CONNECTIONS
    settings["force_proxy"] = FORCE_PROXY

    ses.apply_settings(settings)

    if MAX_DOWNLOAD_SPEED > 0:
        ses.set_download_rate_limit(MAX_DOWNLOAD_SPEED)
    if MAX_UPLOAD_SPEED > 0:
        ses.set_upload_rate_limit(MAX_UPLOAD_SPEED)

    return ses


ses = create_session()
torrent_handles = {}


def load_resume_and_torrents():
    for fastresume_file in DATA_DIR.glob("*.fastresume"):
        info_hash = fastresume_file.stem
        resume_data = fastresume_file.read_bytes()
        torrent_file = DATA_DIR / (info_hash + ".torrent")

        try:
            if torrent_file.exists():
                ti = lt.torrent_info(str(torrent_file))
                atp = lt.read_resume_data(resume_data)
                if not atp.ti:
                    atp.ti = ti

                handle = ses.add_torrent(atp)

                if not AUTO_START:
                    handle.pause()
                    handle.auto_managed(False)

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
    file_path = DATA_DIR / f"{info_hash}.fastresume"
    try:
        file_path.write_bytes(data)
        print(f"Saved fast resume data for {info_hash}")
    except Exception as e:
        print(f"Failed to save fast resume data for {info_hash}: {e}")