# 🧭 Panduan Merancang Fitur & Sistem dari Nol (*Anti Blank-Page Syndrome*)

Kesulitan bingung harus mulai dari mana saat ingin membuat fitur baru dikenal dengan istilah **"Blank Page Syndrome"**. Hal ini sangat normal terjadi bukan karena kurangnya kemampuan coding, melainkan karena otak mencoba memikirkan **UI, API, Database, dan Kode Function secara bersamaan**.

Para engineer berpengalaman mampu merancang sistem dengan rapi karena mereka **memecah masalah secara bertahap (dari gambaran besar ke detail kecil)** menggunakan kerangka berpikir yang sistematis.

Berikut adalah metode langkah demi langkah yang dapat Anda gunakan setiap kali merancang fitur apa pun.

---

## 🔍 5 Langkah Membedah Fitur dari Nol

Jangan langsung membuka code editor untuk menulis kode. Ambil kertas, Notion, atau file teks kosong, lalu ikuti urutan ini:

### Langkah 1: Tulis Cerita Alurnya (*User Flow* & *Edge Cases*)
Sebelum memikirkan tabel atau function, jawab 3 pertanyaan ini dalam bahasa manusia biasa:

1. **Siapa aktornya?** (Misal: User yang sudah login, Admin, atau Publik).
2. **Alur Sukses (*Happy Path*):** Dari awal sampai selesai, apa saja urutannya?
3. **Alur Gagal (*Unhappy Path / Edge Cases*):** Apa saja yang mungkin salah?

> **Contoh Kasus — Fitur "Buat Pesanan (Checkout)":**
> - **Aktor:** User yang sudah login.
> - **Happy Path:** User memilih produk & jumlah $\to$ Klik Checkout $\to$ Sistem cek stok $\to$ Kurangi stok $\to$ Buat catatan order $\to$ Tampilkan detail order.
> - **Edge Cases:** Produk tidak ditemukan, jumlah $\le 0$, stok tidak cukup, atau 2 user membeli barang terakhir secara bersamaan.

---

### Langkah 2: Teknik "Kata Benda vs Kata Kerja"
Cara paling cepat menjembatani ide ke struktur program adalah memisahkan **Kata Benda** dan **Kata Kerja** dari cerita di Langkah 1:

- **Kata Benda = Tabel Database / Struct / Model**
  *(Contoh: `User`, `Product`, `Order`, `OrderItem`)*
- **Kata Kerja = API Endpoints & Functions**
  *(Contoh: `CreateOrder`, `CheckStock`, `DeductStock`, `GetOrderDetail`)*

Jika sebuah fitur belum jelas kata benda dan kata kerjanya, hampir pasti fiturnya masih terlalu abstrak untuk langsung di-coding.

---

### Langkah 3: Rancang Database Terlebih Dahulu (*Data-First Design*)
Mengapa database dulu, bukan API atau UI dulu? Karena **tampilan UI dan format API sering berubah, sedangkan struktur data adalah fondasinya**.

Untuk setiap **Kata Benda**, tentukan 3 hal:
1. **Kolom apa saja yang dibutuhkan?**
2. **Apa relasinya dengan tabel lain?** (*One-to-One*, *One-to-Many*, atau *Many-to-Many*?)
3. **Aturan integritas (*Constraints*):** Mana yang wajib diisi (`NOT NULL`), mana yang tidak boleh kembar (`UNIQUE`), dan mana yang butuh referensi (`FOREIGN KEY`)?

> **Contoh Tabel untuk Fitur Order:**
> - `products` (`id`, `name`, `price`, `stock`)
> - `orders` (`id`, `user_id` $\to$ FK ke `users`, `total_amount`, `status`, `created_at`)
> - `order_items` (`id`, `order_id` $\to$ FK ke `orders`, `product_id` $\to$ FK ke `products`, `quantity`, `price_at_purchase`)
>
> **Pertanyaan Kritis:** *Apakah ada operasi yang mengubah lebih dari 1 tabel sekaligus?*
> **Ya!** Saat checkout, kita mengubah `products.stock`, menambah `orders`, dan menambah `order_items`. Artinya: **wajib pakai Database Transaction (`RunInTx`)**.

---

### Langkah 4: Rancang Kontrak API (*Input $\to$ Output*)
Setelah data jelas, rancang "pintu masuk" aplikasi (API) sebelum menulis logika di dalamnya. Gunakan prinsip **I-P-O (Input – Process – Output)**:

1. **Method & URL:**
   - `POST /api/orders` (membuat data baru)
   - `GET /api/orders/{id}` (mengambil data)
2. **Input (Request Body / Params):**
   - Apa yang **boleh** dikirim oleh frontend? Misal: `product_id` dan `quantity`.
   - **Aturan Keamanan Emas:** *Jangan pernah percaya data sensitif dari frontend.* Jangan biarkan frontend mengirim `user_id`, `price`, atau `total_amount`. `user_id` wajib diambil dari session/JWT di backend, dan `price` wajib diambil dari tabel `products` di database.
3. **Output (Response & Status Code):**
   - Sukses: `201 Created` + data order (`id`, `total_amount`, `status`).
   - Gagal validasi: `400 Bad Request`.
   - Belum login: `401 Unauthorized`.
   - Stok habis: `409 Conflict` atau `422 Unprocessable Entity`.

---

### Langkah 5: Pecah Menjadi Function Kecil (*Pseudocode First*)
Kebingungan menulis function sering terjadi karena Anda mencoba memikirkan seluruh detail sintaks di kepala sekaligus.

Gunakan trik **Tulis Komentar Dulu (*Skeleton Coding*)** di dalam Service Layer:

```go
func (s *orderService) CreateOrder(ctx context.Context, userID uuid.UUID, req CreateOrderRequest) (OrderResponse, error) {
    // 1. Validasi aturan bisnis dasar (misal quantity > 0)
    // 2. Mulai database transaction (RunInTx)
    // 3. Ambil data produk dari DB dan cek apakah stok cukup
    // 4. Hitung total harga di backend (quantity * product.Price)
    // 5. Kurangi stok produk di DB
    // 6. Simpan data ke tabel orders dan order_items
    // 7. Commit transaction, lalu kembalikan OrderResponse
}
```

#### Kapan sebuah kode harus dipecah menjadi function terpisah?
Gunakan 3 indikator sederhana ini:
1. **Beda Layer / Tanggung Jawab:**
   - **Handler:** Hanya urus HTTP (decode JSON, validasi format, panggil Service, tulis HTTP status).
   - **Service:** Hanya urus logika bisnis (hitung harga, cek stok, atur transaksi).
   - **DB / Repository:** Hanya urus query SQL.
2. **Fungsi mulai terlalu panjang / bersarang dalam (*deep nesting*):** Jika di dalam satu fungsi ada `if` di dalam `for` di dalam `if`, pecah bagian dalamnya menjadi helper function kecil dengan nama yang jelas (misal `validateAndReserveStock(...)`).
3. **Ada logika yang dipakai di lebih dari satu tempat:** Ekstrak menjadi 1 fungsi bersama agar tidak duplikat.

---

## 📝 Template Praktis: "1-Page Mini Spec"

Setiap kali Anda ingin membuat fitur baru di project apa pun, salin template singkat ini di catatan Anda dan isi poinnya sebelum mulai menulis kode:

```markdown
## Nama Fitur: [Misal: Ganti Email / Checkout / Komentar]

### 1. Tujuan & Alur (User Flow)
- Aktor: ...
- Alur Sukses: 1) ... -> 2) ... -> 3) ...
- Kemungkinan Gagal (Edge Cases): ...

### 2. Data & Database
- Tabel yang ditambah/diubah: ...
- Relasi antar tabel: ...
- Butuh Database Transaction? (Ya / Tidak, alasannya: ...)

### 3. Kontrak API
- Endpoint: [METHOD] /api/...
- Input dari Client: ...
- Data yang diambil Backend sendiri (JWT/DB): ...
- Output Sukses (Status Code): ...
- Output Error (Status Code): ...

### 4. Daftar Function yang Dibutuhkan
- Handler: ...
- Service: ... (tulis langkah-langkahnya dalam 4-6 poin)
- DB Queries: ...
```

---

## 💡 Latihan Terbaik untuk Melatih Insting Desain

1. **Mulai dari Fitur Kecil, Bukan Langsung Sistem Besar:** Latih diri merancang 1 fitur utuh (misal: *Fitur Tambah Komentar*, lalu *Fitur Bookmark Artikel*, lalu *Fitur Checkout*) menggunakan template 4 bagian di atas.
2. **Biasakan Menggambar Kotak & Panah Sederhana:** Cukup pakai kotak-kotak di kertas (`Frontend -> Handler -> Service -> DB`) dan tuliskan data apa yang bergerak di atas garis panahnya. Jika Anda bisa menggambarnya, Anda pasti bisa meng-coding-nya.
3. **Evaluasi Sebelum Coding (5 Menit):** Tanyakan pada rancangan Anda: *"Kalau langkah ke-3 gagal, apa yang terjadi? Kalau user mengirim data kosong atau nilai minus, di bagian mana ia ditolak?"* — kebiasaan bertanya ini akan langsung menaikkan kualitas desain Anda berkali-kali lipat.
