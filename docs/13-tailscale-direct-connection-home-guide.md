# বাসায় গিয়ে Tailscale direct connection setup

## উদ্দেশ্য

ফোন থেকে server-এর traffic যেন Singapore relay-এর বদলে সরাসরি যায়। Wi-Fi speed বা internet package speed এতে বদলায় না; direct connection সম্ভব হলে delay কমতে পারে।

## ১. আগে WAN IP দেখুন

1. Home Wi-Fi-তে connect করুন। Router-এর gateway/admin address খুলুন (সাধারণত 192.168.0.1; প্রকৃত gateway ব্যবহার করুন)।
2. Internet/WAN status-এর IPv4 দেখুন। এটি screenshot করলে password/secret আড়াল করুন।
3. একই home network থেকে browser-এ public IPv4 দেখে router WAN IPv4-এর সঙ্গে তুলনা করুন।
4. WAN address 10.x.x.x, 172.16–31.x.x, 192.168.x.x বা 100.64–127.x.x হলে upstream private NAT আছে। ISP CGNAT বা দ্বিতীয় router থাকতে পারে। Router forwarding একা যথেষ্ট নাও হতে পারে; ISP-কে public IPv4 চাইতে হবে বা upstream router-ও configure করতে হবে।

Public IP static হওয়া বাধ্যতামূলক নয়। শুধু domain কিনলে relay সমস্যা দূর হবে না।

## ২. Server-এর local IP স্থির রাখুন

Router-এর DHCP Reservation / Address Reservation-এ server-এর Wi-Fi device নির্বাচন করে 192.168.0.149 reserve করুন। এটি 2026-09-26-এ যাচাই করা address; setup করার সময় server-এর বর্তমান address মিলিয়ে নিন। Network interface বা DHCP subnet আন্দাজ করে পরিবর্তন করবেন না।

## ৩. একটি UDP forwarding rule দিন

Router → NAT / Port Forwarding / Virtual Server:

| Field | Value |
|---|---|
| Name | Tailscale server |
| Protocol | UDP |
| External port | 41641 |
| Internal address | 192.168.0.149 |
| Internal port | 41641 |
| Enabled | Yes |

Router admin, SSH, Nextcloud বা FTP port internet-এ খুলবেন না। DMZ প্রয়োজন নেই। Server বর্তমানে UDP 41641-এ listen করে; host firewall inbound packet allow করছে কি না পৃথকভাবে যাচাই করতে হবে। Firewall disable করবেন না।

## ৪. বাইরে থেকে পরীক্ষা

1. ফোনে home Wi-Fi বন্ধ করে mobile data চালু করুন। Tailscale connected রাখুন।
2. Nextcloud খুলুন এবং server থেকে ফোনে `tailscale ping <phone-Tailscale-IP>` পরীক্ষা করান।
3. `via <IP>:<port>` মানে direct; `via DERP(sin)` মানে Singapore relay। প্রথম ping relay হতে পারে, তাই কয়েকবার পরীক্ষা করুন।
4. একই ছবির upload timing তুলনা করুন। Direct হলেই নির্দিষ্ট speed নিশ্চিত নয়; দুই পাশের internet ও server workload প্রভাব ফেলে।

## কাজ না হলে

WAN/public IP তুলনা, double NAT, ISP CGNAT, host firewall ও ফোনের network UDP restrictions পরীক্ষা করুন। Router UPnP/NAT-PMP দেখা গেছে, কিন্তু সেটি successful public inbound mapping-এর প্রমাণ নয়। সব device-এর route বা Tailscale exit node পরিবর্তন করার প্রয়োজন নেই। প্রয়োজন হলে reachable peer relay নিয়ে আলোচনা করুন।

Rollback: শুধু নতুন UDP forwarding rule disable/remove করুন; অন্য network settings অপরিবর্তিত রাখুন।

Official references: [Firewall ports](https://tailscale.com/docs/reference/faq/firewall-ports), [Connection types](https://tailscale.com/docs/reference/connection-types).
