E-Commerce Platform (Go + PostgreSQL + JS Frontend)

A production-ready, modular e-commerce platform built with Go, PostgreSQL, and a lightweight vanilla JavaScript frontend.
The system includes user authentication with email verification, password-based login, persistent shopping carts, checkout processing, and transactional email notifications.

⸻

🔥 Features

Authentication
	•	Email-based registration with verification codes
	•	Password hashing using bcrypt
	•	Secure JWT issuance and storage via HttpOnly cookies
	•	Full login/logout and session validation endpoints

Products
	•	Product listing served via a REST endpoint
	•	Clean JSON schema and image handling via static assets

Shopping Cart
	•	Auth-required persistent cart per user
	•	Add, update, remove items (quantity-based)
	•	Strong DB consistency using GORM models with FK constraints

Checkout
	•	Calculates order totals server-side (no trust in client data)
	•	Sends HTML-based “Order Confirmation” email
	•	Clears the cart atomically after checkout

⸻

🧱 Tech Stack
	•	Go 1.23+
	•	PostgreSQL 14+
	•	Gin Web Framework
	•	GORM ORM
	•	Nginx Reverse Proxy + HTTPS
	•	systemd service for backend runtime
	•	Vanilla JS frontend, no frameworks
	•	MailHog / SMTP for email delivery

  ----------------------------------------------------------------------------------------------

Cakarokko E-Commerce

Go + Gin + PostgreSQL + JWT + SMTP Email kullanılarak geliştirilmiş production-ready bir e-ticaret uygulaması.

Bu proje; kullanıcı kaydı, e-posta doğrulama, şifre ile giriş, ürün listeleme, sepet yönetimi ve checkout işleminde sipariş özeti maili gönderme gibi tüm temel e-ticaret fonksiyonlarını içerir.
Frontend tamamen HTML/JS ile yapılmış minimal ve hızlı bir arayüzdür.

⸻

🚀 Özellikler

🔐 Kimlik Doğrulama
	•	Email + Şifre ile kayıt
	•	SMTP ile doğrulama kodu gönderimi
	•	Bcrypt ile güvenli şifre hashleme
	•	Doğrulanmamış kullanıcıya login izni yok
	•	JWT + HttpOnly cookie ile giriş
	•	Logout desteği

🛒 Sepet Sistemi
	•	Ürün sepete ekle
	•	Ürün adetini artır/azalt
	•	Adet 0 olursa otomatik sil
	•	Sepeti görüntüleme
	•	Checkout → satın alma maili gönderimi
	•	Checkout sonrası sepetin boşaltılması

📦 Ürün Yönetimi
	•	Ürün listeleme (/api/products)
	•	Görsel, isim, fiyat gösterimi
	•	Frontend grid tasarımı

✉ Mail Gönderimi
	•	Kullanıcı doğrulama kodu
	•	Sipariş özeti e-maili
	•	HTML formatlı temiz mail layout

🖥 Frontend
	•	Vanilla JS + HTML5
	•	Minimal responsive tasarım
	•	Sepet butonları: artır/azalt/sil
	•	Kayıt / Doğrulama / Giriş ekranları
	•	Login sonrası kullanıcı email gösterimi
