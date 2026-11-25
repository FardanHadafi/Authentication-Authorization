# 🧠 Mengubah "Tahu" Menjadi "Bisa": Mengatasi *Illusion of Competence*

Kondisi saat Anda paham ketika membaca kode, tetapi mendadak "ngeblank" ketika harus menulisnya dari layar kosong, bukanlah tanda ketidakmampuan—melainkan mekanisme biologis otak manusia yang sudah diteliti secara luas dalam psikologi kognitif.

Secara ilmiah, fenomena ini terjadi karena perbedaan antara **Recognition (Pengenalan)** dan **Recall (Pemanggilan Ulang)**:

- **Saat membaca kode:** Anda menggunakan **Recognition Memory** (*memori pasif*). Otak melihat sintaks yang masuk akal, lalu berkata *"Oh iya, ngerti!"*. Proses ini mudah dan memicu rasa puas, padahal hanya membuktikan bahwa Anda bisa **membaca**, bukan **memproduksi**.
- **Saat menulis kode dari nol:** Anda menggunakan **Recall & Procedural Memory** (*memori aktif*). Anda harus memanggil sintaks, struktur logika, urutan eksekusi, dan edge cases dari ingatan kosong sambil menahannya di **Working Memory** (yang kapasitasnya secara ilmiah sangat sempit, hanya sekitar 4–7 potongan informasi sekaligus).

Ketika *Working Memory* kewalahan, otak langsung mengalami *"freeze"* atau ngeblank.

Berikut adalah **6 metode berbasis sains** untuk menghancurkan *Illusion of Competence* dan mengubah "tahu" menjadi "bisa":

---

## 1. *Retrieval Practice*: Teknik "Baca $\to$ Tutup $\to$ Tulis"

- **Dasar Sains:** Studi terkenal oleh **Roediger & Karpicke (*Testing Effect*)** membuktikan bahwa memaksa otak menarik informasi dari ingatan (*active recall*) menghasilkan penguasaan jangka panjang 2–3x lebih kuat daripada membaca ulang atau menonton tutorial berulang kali.
- **Cara Praktik di Coding:**
  1. Pilih **1 fungsi kecil** saja (15–30 baris, misal fungsi `Register` atau `GetProfile`). Baca dan pahami alurnya.
  2. **Tutup file tersebut sepenuhnya** (jangan ditaruh di *split screen*).
  3. Buka file kosong, lalu tulis ulang logika yang sama dari nol.
  4. **Saat Anda ngeblank:** Jangan langsung mengintip! Paksa otak berpikir 1–2 menit. Jika benar-benar buntu, intip **1 bagian kecil saja** yang lupa, tutup lagi, lalu lanjutkan menulis.

---

## 2. *Desirable Difficulty*: Jangan Takut Merasa "Bodoh"

- **Dasar Sains:** Psikolog kognitif **Robert Bjork** menemukan konsep ***Desirable Difficulty*** (kesulitan yang diinginkan). Ketika belajar terasa nyaman dan mulus (seperti membaca kode AI yang rapi), otak hampir tidak membentuk jalur saraf baru (*myelination*). Justru saat Anda merasa frustrasi, ngeblank, dan berpikir keras mencari solusi, di detik itulah sinapsis saraf otak sedang tumbuh dan menguat.
- **Cara Praktik:** Terapkan **Aturan Timer 15 Menit**. Saat macet atau muncul error, tahan diri untuk tidak langsung bertanya ke AI selama 15 menit. Baca pesan error-nya, telusuri baris kodenya, coba perbaiki sendiri. Jika setelah 15 menit masih buntu, barulah minta **petunjuk (*hint*)**, bukan kode jadi.

---

## 3. *Self-Explanation* & *Feynman Technique*

- **Dasar Sains:** Penelitian oleh **Michelene Chi (*Self-Explanation Effect*)** menunjukkan bahwa pelajar yang menjelaskan setiap langkah dengan bahasanya sendiri mampu mendeteksi lubang pemahaman (*knowledge gap*) yang tersembunyi saat membaca pasif.
- **Cara Praktik:** Sebelum menulis kode, tulis dulu langkahnya dengan bahasa manusia biasa dalam bentuk komentar:
  ```go
  // 1. Ambil JSON dari request, kalau format salah kembalikan 400
  // 2. Cek ke DB apakah email sudah ada
  // 3. Kalau belum ada, hash password lalu simpan
  ```
  Jika Anda tidak bisa menjelaskan *"mengapa baris ini harus ada di sini dan bukan di bawahnya"*, artinya Anda belum benar-benar paham—baru sekadar akrab melihatnya.

---

## 4. *Chunking* & *Micro-Drills* (Mengatasi *Cognitive Overload*)

- **Dasar Sains:** Berdasarkan **Cognitive Load Theory (John Sweller)**, pemula ngeblank karena mencoba memikirkan terlalu banyak hal sekaligus (HTTP + JSON + Validasi + SQL + Error Handling). Engineer senior tidak punya otak yang lebih besar; mereka hanya sudah menggabungkan pola berulang menjadi satu **"Chunk"** (paket ingatan).
- **Cara Praktik (Latihan Mikro 15 Menit):** Jangan langsung latihan membangun sistem Login lengkap. Pecah dan latih per bagian kecil secara berulang tanpa *copy-paste*:
  - **Latihan 1:** Buat hanya HTTP Handler yang menerima JSON, validasi, dan return JSON (ulangi 3 kali untuk endpoint berbeda).
  - **Latihan 2:** Buat hanya fungsi Service yang menerima input, memanggil 1 query DB, dan menangani error-nya.
  - **Latihan 3:** Baru gabungkan Handler + Service.

---

## 5. *Transfer of Learning*: Ubah Konteksnya (Jangan Hanya Meniru)

- **Dasar Sains:** Jika Anda hanya mengulang contoh yang persis sama, otak hanya menghafal (*rote memorization*). Pemahaman sejati baru terbentuk ketika Anda memaksa otak menerapkan pola yang sama pada kasus yang berbeda.
- **Cara Praktik:** Setelah berhasil menulis ulang endpoint `POST /register` tanpa melihat catatan, langsung tantang diri Anda membuat endpoint `POST /articles` atau `POST /products` menggunakan pola arsitektur yang sama (`DTO -> Handler -> Service -> DB`). Begitu Anda bisa memodifikasinya untuk kasus baru, skill itu resmi menjadi milik Anda.

---

## 6. Ubah Cara Menggunakan AI: Dari "Tukang Ketik" Menjadi "Pelatih"

Jika Anda menggunakan AI dengan prompt *"Buatkan fitur X"*, AI melakukan 100% proses *Recall*, dan Anda hanya kebagian proses *Recognition* (membaca hasilnya).

Ubah cara interaksinya menjadi 3 pola ini:

1. **Minta Soal, Bukan Jawaban:**
   > *"Saya mau latihan implementasi handler & service di Go. Jangan tuliskan kodenya. Beri saya 1 studi kasus kecil beserta requirement input-outputnya, biar saya yang coding dari nol."*
2. **Minta *Code Review*, Bukan *Generate*:**
   > *"Ini kode yang saya tulis sendiri (meskipun masih berantakan/error). Jangan langsung tulis ulang semuanya; tunjukkan bagian mana yang salah logika atau kurang aman dan jelaskan alasannya."*
3. **Minta *Progressive Hint*:**
   > *"Saya stuck di error ini sudah 15 menit. Beri saya petunjuk arahnya saja dulu, jangan langsung kasih kode solusinya."*

---

## ⏱️ Rutinitas Praktis 45 Menit Setiap Hari

Jika Anda mempraktikkan pola ini secara konsisten selama 2–3 minggu:

- **10 menit:** Baca 1 fungsi/konsep kecil dan pahami alurnya.
- **25 menit:** Tutup referensi, buka layar kosong, tulis ulang lalu modifikasi untuk kasus baru (nikmati proses mikir kerasnya).
- **10 menit:** Bandingkan kode Anda dengan referensi asli atau minta AI me-review kode buatan Anda.

Rasa "ngeblank" di awal adalah harga masuk yang wajar untuk setiap skill nyata. Semakin sering Anda memaksa otak keluar dari zona nyaman membaca menuju zona menulis mandiri, semakin cepat jeda antara *"tahu"* dan *"bisa"* akan menghilang.
