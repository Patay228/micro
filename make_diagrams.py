# -*- coding: utf-8 -*-
"""
make_diagrams.py
Скрипт генерации исходных файлов draw.io и PNG-диаграмм архитектуры и последовательности
для Лабораторной работы №3 по микросервисам (Информационная система цирка).
"""

import os
import matplotlib
import matplotlib.pyplot as plt
import matplotlib.patches as patches

# Настройка шрифта для корректного отображения кириллицы
plt.rcParams['font.family'] = 'Segoe UI'
plt.rcParams['font.size'] = 10

# -----------------------------------------------------------------------------
# 1. Генерация docs/architecture.drawio
# -----------------------------------------------------------------------------
drawio_architecture = """<mxfile host="app.diagrams.net" modified="2026-10-13T14:00:00.000Z" agent="Antigravity" version="24.0.0" type="device">
  <diagram id="circus-arch" name="Архитектура цирковой микросервисной системы">
    <mxGraphModel dx="1400" dy="900" grid="1" gridSize="10" guides="1" tooltips="1" connect="1" arrows="1" fold="1" page="1" pageScale="1" pageWidth="1400" pageHeight="950" math="0" shadow="0">
      <root>
        <mxCell id="0" />
        <mxCell id="1" parent="0" />
        
        <!-- Clients -->
        <mxCell id="client" value="Клиент: Веб-сайт и Мобильное приложение&#xa;(Зрители, Администраторы, Маркетологи)" style="rounded=1;whiteSpace=wrap;html=1;fillColor=#e1f5fe;strokeColor=#0288d1;fontStyle=1;fontSize=13;" vertex="1" parent="1">
          <mxGeometry x="460" y="30" width="360" height="60" as="geometry" />
        </mxCell>
        
        <!-- API Gateway -->
        <mxCell id="gateway" value="API Gateway (Spring Cloud Gateway)&#xa;Маршрутизация /api/v1/*, Rate Limiting, Проверка JWT" style="rounded=1;whiteSpace=wrap;html=1;fillColor=#bbdefb;strokeColor=#1976d2;fontStyle=1;fontSize=12;" vertex="1" parent="1">
          <mxGeometry x="460" y="140" width="360" height="60" as="geometry" />
        </mxCell>
        
        <!-- Keycloak -->
        <mxCell id="keycloak" value="Keycloak (IAM)&#xa;OAuth 2.0 / OpenID Connect&#xa;Аутентификация и токены" style="rounded=1;whiteSpace=wrap;html=1;fillColor=#f5f5f5;strokeColor=#616161;fontStyle=1;fontSize=11;" vertex="1" parent="1">
          <mxGeometry x="880" y="140" width="200" height="60" as="geometry" />
        </mxCell>
        
        <!-- Services -->
        <mxCell id="catalog" value="catalog-service&#xa;Афиша, программы, сеансы, цены&#xa;PostgreSQL + Redis (кэш)" style="rounded=1;whiteSpace=wrap;html=1;fillColor=#e8f5e9;strokeColor=#388e3c;fontStyle=1;fontSize=11;" vertex="1" parent="1">
          <mxGeometry x="40" y="270" width="220" height="85" as="geometry" />
        </mxCell>
        
        <mxCell id="booking" value="booking-service&#xa;Бронь мест (15 мин), билеты, сага&#xa;PostgreSQL + Redis (hold)" style="rounded=1;whiteSpace=wrap;html=1;fillColor=#fff3e0;strokeColor=#f57c00;fontStyle=1;fontSize=11;" vertex="1" parent="1">
          <mxGeometry x="320" y="270" width="240" height="85" as="geometry" />
        </mxCell>
        
        <mxCell id="loyalty" value="loyalty-service&#xa;Баллы, ранги, скидки, правила&#xa;PostgreSQL" style="rounded=1;whiteSpace=wrap;html=1;fillColor=#f3e5f5;strokeColor=#7b1fa2;fontStyle=1;fontSize=11;" vertex="1" parent="1">
          <mxGeometry x="620" y="270" width="220" height="85" as="geometry" />
        </mxCell>
        
        <mxCell id="payment" value="payment-service&#xa;Платежи, эквайринг, чеки, возвраты&#xa;PostgreSQL" style="rounded=1;whiteSpace=wrap;html=1;fillColor=#e0f2f1;strokeColor=#00796b;fontStyle=1;fontSize=11;" vertex="1" parent="1">
          <mxGeometry x="900" y="270" width="230" height="85" as="geometry" />
        </mxCell>
        
        <mxCell id="notification" value="notification-service&#xa;Билеты с QR, email/SMS оповещения&#xa;MongoDB (шаблоны и история)" style="rounded=1;whiteSpace=wrap;html=1;fillColor=#fce4ec;strokeColor=#c2185b;fontStyle=1;fontSize=11;" vertex="1" parent="1">
          <mxGeometry x="480" y="620" width="310" height="80" as="geometry" />
        </mxCell>
        
        <!-- Broker Kafka -->
        <mxCell id="kafka" value="Apache Kafka (Брокер сообщений)&#xa;Топики: circus.payments.events, circus.bookings.events, circus.refunds.events, circus.loyalty.events" style="rounded=1;whiteSpace=wrap;html=1;fillColor=#eceff1;strokeColor=#455a64;fontStyle=1;fontSize=12;" vertex="1" parent="1">
          <mxGeometry x="140" y="460" width="1000" height="65" as="geometry" />
        </mxCell>
        
        <!-- External Systems -->
        <mxCell id="bank" value="Банковский эквайринг&#xa;(Платёжный шлюз: карты, СБП)" style="rounded=1;whiteSpace=wrap;html=1;fillColor=#efebe9;strokeColor=#5d4037;fontStyle=1;fontSize=11;" vertex="1" parent="1">
          <mxGeometry x="1190" y="270" width="180" height="85" as="geometry" />
        </mxCell>
        
        <mxCell id="sms_email" value="SMS / Email Провайдер&#xa;(Внешний транспорт)" style="rounded=1;whiteSpace=wrap;html=1;fillColor=#efebe9;strokeColor=#5d4037;fontStyle=1;fontSize=11;" vertex="1" parent="1">
          <mxGeometry x="870" y="620" width="180" height="80" as="geometry" />
        </mxCell>
        
        <!-- Edges -->
        <mxCell id="e_client_gw" value="HTTPS" style="edgeStyle=orthogonalEdgeStyle;rounded=0;html=1;entryX=0.5;entryY=0;strokeWidth=2;" edge="1" source="client" target="gateway" parent="1" />
        <mxCell id="e_gw_keycloak" value="проверка JWT" style="edgeStyle=orthogonalEdgeStyle;rounded=0;html=1;strokeWidth=1;" edge="1" source="gateway" target="keycloak" parent="1" />
        
        <mxCell id="e_gw_catalog" value="GET /api/v1/shows, sessions" style="edgeStyle=orthogonalEdgeStyle;rounded=0;html=1;strokeWidth=1.5;" edge="1" source="gateway" target="catalog" parent="1" />
        <mxCell id="e_gw_booking" value="POST /api/v1/bookings" style="edgeStyle=orthogonalEdgeStyle;rounded=0;html=1;strokeWidth=1.5;" edge="1" source="gateway" target="booking" parent="1" />
        <mxCell id="e_gw_loyalty" value="GET /api/v1/loyalty/me" style="edgeStyle=orthogonalEdgeStyle;rounded=0;html=1;strokeWidth=1.5;" edge="1" source="gateway" target="loyalty" parent="1" />
        
        <mxCell id="e_booking_catalog" value="GET /internal/v1/sessions/{id}" style="edgeStyle=orthogonalEdgeStyle;rounded=0;html=1;strokeWidth=1.5;" edge="1" source="booking" target="catalog" parent="1" />
        <mxCell id="e_booking_loyalty" value="POST /internal/v1/loyalty/reservations" style="edgeStyle=orthogonalEdgeStyle;rounded=0;html=1;strokeWidth=1.5;" edge="1" source="booking" target="loyalty" parent="1" />
        <mxCell id="e_booking_payment" value="POST /internal/v1/payments" style="edgeStyle=orthogonalEdgeStyle;rounded=0;html=1;strokeWidth=1.5;" edge="1" source="booking" target="payment" parent="1" />
        
        <mxCell id="e_payment_bank" value="REST API" style="edgeStyle=orthogonalEdgeStyle;rounded=0;html=1;strokeWidth=1.5;" edge="1" source="payment" target="bank" parent="1" />
        <mxCell id="e_bank_payment" value="Webhook" style="edgeStyle=orthogonalEdgeStyle;rounded=0;html=1;strokeWidth=1.5;" edge="1" source="bank" target="payment" parent="1" />
        
        <mxCell id="e_payment_kafka" value="payment.completed, payment.failed" style="edgeStyle=orthogonalEdgeStyle;rounded=0;html=1;strokeWidth=1.5;dashed=1;" edge="1" source="payment" target="kafka" parent="1" />
        <mxCell id="e_kafka_booking" value="payment.completed, payment.failed" style="edgeStyle=orthogonalEdgeStyle;rounded=0;html=1;strokeWidth=1.5;dashed=1;" edge="1" source="kafka" target="booking" parent="1" />
        
        <mxCell id="e_booking_kafka" value="booking.confirmed, booking.expired, refund.requested" style="edgeStyle=orthogonalEdgeStyle;rounded=0;html=1;strokeWidth=1.5;dashed=1;" edge="1" source="booking" target="kafka" parent="1" />
        <mxCell id="e_kafka_loyalty" value="booking.confirmed, refund.requested" style="edgeStyle=orthogonalEdgeStyle;rounded=0;html=1;strokeWidth=1.5;dashed=1;" edge="1" source="kafka" target="loyalty" parent="1" />
        <mxCell id="e_loyalty_kafka" value="loyalty.tier.upgraded, loyalty.points.accrued" style="edgeStyle=orthogonalEdgeStyle;rounded=0;html=1;strokeWidth=1.5;dashed=1;" edge="1" source="loyalty" target="kafka" parent="1" />
        
        <mxCell id="e_kafka_notif" value="booking.confirmed, booking.expired, refund.completed, loyalty.tier.upgraded" style="edgeStyle=orthogonalEdgeStyle;rounded=0;html=1;strokeWidth=1.5;dashed=1;" edge="1" source="kafka" target="notification" parent="1" />
        <mxCell id="e_notif_ext" value="Отправка билетов" style="edgeStyle=orthogonalEdgeStyle;rounded=0;html=1;strokeWidth=1.5;" edge="1" source="notification" target="sms_email" parent="1" />
      </root>
    </mxGraphModel>
  </diagram>
</mxfile>"""

with open('docs/architecture.drawio', 'w', encoding='utf-8') as f:
    f.write(drawio_architecture)
print('Generated docs/architecture.drawio')

# -----------------------------------------------------------------------------
# 2. Генерация docs/sequence-diagrams.drawio
# -----------------------------------------------------------------------------
drawio_sequences = """<mxfile host="app.diagrams.net" modified="2026-10-13T14:00:00.000Z" agent="Antigravity" version="24.0.0" type="device">
  <diagram id="seq-success" name="1. Успешная покупка билета (Happy Path)">
    <mxGraphModel dx="1400" dy="900" grid="1" gridSize="10" guides="1" tooltips="1" connect="1" arrows="1" fold="1" page="1" pageScale="1" pageWidth="1200" pageHeight="900">
      <root>
        <mxCell id="0" />
        <mxCell id="1" parent="0" />
        <!-- Sequence lifelines -->
        <mxCell id="ll_client" value="Клиент" style="shape=umlLifeline;perimeter=lifelinePerimeter;whiteSpace=wrap;html=1;container=1;collapsible=0;recursiveResize=0;outlineConnect=0;" vertex="1" parent="1">
          <mxGeometry x="60" y="40" width="100" height="600" as="geometry" />
        </mxCell>
        <mxCell id="ll_gw" value="API Gateway" style="shape=umlLifeline;perimeter=lifelinePerimeter;whiteSpace=wrap;html=1;container=1;collapsible=0;recursiveResize=0;outlineConnect=0;" vertex="1" parent="1">
          <mxGeometry x="190" y="40" width="100" height="600" as="geometry" />
        </mxCell>
        <mxCell id="ll_booking" value="booking-service" style="shape=umlLifeline;perimeter=lifelinePerimeter;whiteSpace=wrap;html=1;container=1;collapsible=0;recursiveResize=0;outlineConnect=0;" vertex="1" parent="1">
          <mxGeometry x="320" y="40" width="110" height="600" as="geometry" />
        </mxCell>
        <mxCell id="ll_catalog" value="catalog-service" style="shape=umlLifeline;perimeter=lifelinePerimeter;whiteSpace=wrap;html=1;container=1;collapsible=0;recursiveResize=0;outlineConnect=0;" vertex="1" parent="1">
          <mxGeometry x="460" y="40" width="110" height="600" as="geometry" />
        </mxCell>
        <mxCell id="ll_loyalty" value="loyalty-service" style="shape=umlLifeline;perimeter=lifelinePerimeter;whiteSpace=wrap;html=1;container=1;collapsible=0;recursiveResize=0;outlineConnect=0;" vertex="1" parent="1">
          <mxGeometry x="600" y="40" width="110" height="600" as="geometry" />
        </mxCell>
        <mxCell id="ll_payment" value="payment-service" style="shape=umlLifeline;perimeter=lifelinePerimeter;whiteSpace=wrap;html=1;container=1;collapsible=0;recursiveResize=0;outlineConnect=0;" vertex="1" parent="1">
          <mxGeometry x="740" y="40" width="110" height="600" as="geometry" />
        </mxCell>
        <mxCell id="ll_kafka" value="Apache Kafka" style="shape=umlLifeline;perimeter=lifelinePerimeter;whiteSpace=wrap;html=1;container=1;collapsible=0;recursiveResize=0;outlineConnect=0;" vertex="1" parent="1">
          <mxGeometry x="880" y="40" width="110" height="600" as="geometry" />
        </mxCell>
        <mxCell id="ll_notif" value="notification" style="shape=umlLifeline;perimeter=lifelinePerimeter;whiteSpace=wrap;html=1;container=1;collapsible=0;recursiveResize=0;outlineConnect=0;" vertex="1" parent="1">
          <mxGeometry x="1020" y="40" width="110" height="600" as="geometry" />
        </mxCell>
      </root>
    </mxGraphModel>
  </diagram>
</mxfile>"""

with open('docs/sequence-diagrams.drawio', 'w', encoding='utf-8') as f:
    f.write(drawio_sequences)
print('Generated docs/sequence-diagrams.drawio')

# -----------------------------------------------------------------------------
# 3. Рендеринг PNG: docs/architecture.png
# -----------------------------------------------------------------------------
def render_architecture_png():
    fig, ax = plt.subplots(figsize=(15, 10.5), dpi=200)
    ax.set_xlim(0, 15)
    ax.set_ylim(0, 11)
    ax.axis('off')
    
    # Title
    ax.text(7.5, 10.4, "Архитектура микросервисной системы цирка", 
            ha='center', va='center', fontsize=18, fontweight='bold', color='#1A237E')
    ax.text(7.5, 10.05, "Лабораторная работа №3 • Декомпозиция, базы данных, Kafka и контракты API", 
            ha='center', va='center', fontsize=11, color='#546E7A')

    # Helper for boxes
    def draw_box(x, y, w, h, title, subtitle, bg_color, border_color, title_color='#000000'):
        rect = patches.FancyBboxPatch((x, y), w, h, boxstyle="round,pad=0.1,rounding_size=0.15",
                                      facecolor=bg_color, edgecolor=border_color, linewidth=2)
        ax.add_patch(rect)
        ax.text(x + w/2, y + h*0.65, title, ha='center', va='center', fontsize=11, fontweight='bold', color=title_color)
        ax.text(x + w/2, y + h*0.28, subtitle, ha='center', va='center', fontsize=8.5, color='#37474F')
        return (x, y, w, h)

    # 1. Client
    draw_box(4.5, 8.8, 6.0, 0.9, 
             "Клиент: Веб-сайт и Мобильное приложение", 
             "Зрители, Администраторы афиши, Маркетологи (HTTPS)", 
             '#E1F5FE', '#0288D1', '#01579B')

    # 2. Gateway & Keycloak
    draw_box(4.5, 7.3, 4.2, 0.9, 
             "API Gateway (Spring Cloud Gateway)", 
             "Точка входа /api/v1/* • Rate Limiter • JWT Validation", 
             '#BBDEFB', '#1976D2', '#0D47A1')

    draw_box(9.2, 7.3, 3.2, 0.9, 
             "Keycloak (IAM Service)", 
             "OAuth 2.0 / OIDC • Авторизация и JWT токены", 
             '#ECEFF1', '#607D8B', '#263238')

    # Arrows Client -> Gateway and Gateway <-> Keycloak
    ax.annotate("", xy=(6.6, 8.2), xytext=(6.6, 8.8),
                arrowprops=dict(arrowstyle="->", color="#0D47A1", lw=2))
    ax.text(6.7, 8.5, "HTTPS / REST", fontsize=8.5, color="#0D47A1", fontweight='bold')

    ax.annotate("", xy=(9.2, 7.75), xytext=(8.7, 7.75),
                arrowprops=dict(arrowstyle="<->", color="#455A64", lw=1.5, ls="--"))
    ax.text(8.95, 8.0, "проверка JWT", fontsize=8, ha='center', color="#37474F")

    # 3. Business Services
    # Catalog
    draw_box(0.6, 5.0, 2.6, 1.3, 
             "catalog-service", 
             "Афиша, программы, сеансы\nБД: PostgreSQL (catalog_db)\nКэш: Redis (сеансы)", 
             '#E8F5E9', '#388E3C', '#1B5E20')

    # Booking
    draw_box(3.6, 5.0, 2.7, 1.3, 
             "booking-service", 
             "Бронь мест, билеты, Saga\nБД: PostgreSQL (booking_db)\nБлокировки: Redis (15 мин)", 
             '#FFF3E0', '#F57C00', '#E65100')

    # Loyalty
    draw_box(6.7, 5.0, 2.6, 1.3, 
             "loyalty-service", 
             "Баллы, ранги, правила\nБД: PostgreSQL (loyalty_db)\nТранзакции начислений", 
             '#F3E5F5', '#7B1FA2', '#4A148C')

    # Payment
    draw_box(9.7, 5.0, 2.7, 1.3, 
             "payment-service", 
             "Платежи, эквайринг, возвраты\nБД: PostgreSQL (payment_db)\nЧеки и аудит", 
             '#E0F2F1', '#00796B', '#004D40')

    # External Bank
    draw_box(12.7, 5.0, 2.0, 1.3, 
             "Банковский эквайринг", 
             "Внешний шлюз\nКарты, СБП\nПлатежная страница", 
             '#EFEBE9', '#5D4037', '#3E2723')

    # Gateway -> Services lines
    # GW -> Catalog
    ax.annotate("", xy=(1.9, 6.3), xytext=(4.7, 7.3),
                arrowprops=dict(arrowstyle="->", color="#2E7D32", lw=1.5))
    ax.text(2.6, 6.85, "GET /api/v1/shows", fontsize=8, color="#2E7D32", rotation=22)

    # GW -> Booking
    ax.annotate("", xy=(4.95, 6.3), xytext=(5.6, 7.3),
                arrowprops=dict(arrowstyle="->", color="#E65100", lw=1.5))
    ax.text(5.4, 6.75, "POST /api/v1/bookings", fontsize=8, color="#E65100", rotation=-45)

    # GW -> Loyalty
    ax.annotate("", xy=(7.8, 6.3), xytext=(7.2, 7.3),
                arrowprops=dict(arrowstyle="->", color="#6A1B9A", lw=1.5))
    ax.text(7.6, 6.8, "GET /api/v1/loyalty/me", fontsize=8, color="#6A1B9A", rotation=45)

    # Internal Synchronous REST calls between services
    # Booking -> Catalog
    ax.annotate("", xy=(3.2, 5.75), xytext=(3.6, 5.75),
                arrowprops=dict(arrowstyle="->", color="#1B5E20", lw=1.5))
    ax.text(3.4, 6.0, "GET /internal/v1/sessions/{id}", fontsize=7.5, ha='center', color="#1B5E20", fontweight='bold')

    # Booking -> Loyalty
    ax.annotate("", xy=(6.7, 5.75), xytext=(6.3, 5.75),
                arrowprops=dict(arrowstyle="->", color="#4A148C", lw=1.5))
    ax.text(6.5, 6.0, "POST /internal/v1/loyalty/reservations", fontsize=7.5, ha='center', color="#4A148C", fontweight='bold')

    # Booking -> Payment
    ax.annotate("", xy=(9.7, 5.4), xytext=(6.3, 5.4),
                arrowprops=dict(arrowstyle="->", color="#004D40", lw=1.5))
    ax.text(8.0, 5.15, "POST /internal/v1/payments (сумма за вычетом баллов)", fontsize=8, ha='center', color="#004D40", fontweight='bold')

    # Payment <-> Bank
    ax.annotate("", xy=(12.7, 5.85), xytext=(12.4, 5.85),
                arrowprops=dict(arrowstyle="->", color="#3E2723", lw=1.5))
    ax.text(12.55, 6.05, "Создать сессию", fontsize=7.5, ha='center', color="#3E2723")

    ax.annotate("", xy=(12.4, 5.3), xytext=(12.7, 5.3),
                arrowprops=dict(arrowstyle="->", color="#C62828", lw=1.5))
    ax.text(12.55, 5.1, "Webhook (оплата)", fontsize=7.5, ha='center', color="#C62828")

    # 4. Message Broker (Apache Kafka)
    draw_box(1.5, 3.0, 11.0, 1.0, 
             "Apache Kafka (Распределённый брокер событий)", 
             "Топики: payments (payment.completed/failed) • bookings (booking.confirmed/expired) • refunds • loyalty", 
             '#ECEFF1', '#455A64', '#263238')

    # Payment -> Kafka
    ax.annotate("", xy=(10.5, 4.0), xytext=(10.5, 5.0),
                arrowprops=dict(arrowstyle="->", color="#00796B", lw=1.8, ls="--"))
    ax.text(10.6, 4.5, "payment.completed\npayment.failed", fontsize=8, color="#00796B")

    # Kafka -> Booking
    ax.annotate("", xy=(4.5, 5.0), xytext=(4.5, 4.0),
                arrowprops=dict(arrowstyle="->", color="#F57C00", lw=1.8, ls="--"))
    ax.text(3.7, 4.4, "payment.*\n(подтверждение)", fontsize=8, color="#F57C00")

    # Booking -> Kafka
    ax.annotate("", xy=(5.5, 4.0), xytext=(5.5, 5.0),
                arrowprops=dict(arrowstyle="->", color="#F57C00", lw=1.8, ls="--"))
    ax.text(5.6, 4.5, "booking.confirmed\nbooking.expired", fontsize=8, color="#E65100")

    # Kafka -> Loyalty
    ax.annotate("", xy=(7.5, 5.0), xytext=(7.5, 4.0),
                arrowprops=dict(arrowstyle="->", color="#7B1FA2", lw=1.8, ls="--"))
    ax.text(7.6, 4.4, "booking.confirmed\n(кэшбэк баллы)", fontsize=8, color="#7B1FA2")

    # Loyalty -> Kafka
    ax.annotate("", xy=(8.4, 4.0), xytext=(8.4, 5.0),
                arrowprops=dict(arrowstyle="->", color="#7B1FA2", lw=1.8, ls="--"))
    ax.text(8.5, 4.7, "loyalty.tier.upgraded", fontsize=7.5, color="#7B1FA2")

    # 5. Notification Service
    draw_box(4.5, 1.2, 4.5, 1.1, 
             "notification-service", 
             "Отправка билетов с QR, чеков и напоминаний\nБД: MongoDB (notification_db, шаблоны и журнал)", 
             '#FCE4EC', '#C2185B', '#880E4F')

    # External Notification Provider (SMS / Email)
    draw_box(10.0, 1.2, 3.2, 1.1, 
             "SMS / Email Провайдер", 
             "Внешний сервис рассылок\nSMTP, SMPP, Telegram Bot API", 
             '#EFEBE9', '#5D4037', '#3E2723')

    # Kafka -> Notification
    ax.annotate("", xy=(6.5, 2.3), xytext=(6.5, 3.0),
                arrowprops=dict(arrowstyle="->", color="#C2185B", lw=1.8, ls="--"))
    ax.text(6.6, 2.65, "booking.confirmed / booking.expired / refund.completed", fontsize=8, color="#C2185B")

    # Notification -> Provider
    ax.annotate("", xy=(10.0, 1.75), xytext=(9.0, 1.75),
                arrowprops=dict(arrowstyle="->", color="#880E4F", lw=1.5))
    ax.text(9.5, 1.9, "Отправка писем и SMS", fontsize=8, ha='center', color="#880E4F")

    # Legend
    leg_rect = patches.FancyBboxPatch((0.6, 0.2), 13.8, 0.7, boxstyle="round,pad=0.05,rounding_size=0.1",
                                     facecolor='#FAFAFA', edgecolor='#BDBDBD', linewidth=1)
    ax.add_patch(leg_rect)
    ax.text(1.2, 0.55, "Условные обозначения:", fontsize=9, fontweight='bold', color='#424242')
    
    # Solid line example
    ax.plot([3.5, 4.3], [0.55, 0.55], color='#1976D2', lw=2)
    ax.text(4.4, 0.55, "Синхронный вызов (REST / HTTPS)", fontsize=8.5, va='center', color='#212121')

    # Dashed line example
    ax.plot([7.8, 8.6], [0.55, 0.55], color='#D81B60', lw=2, ls='--')
    ax.text(8.7, 0.55, "Асинхронное событие (Apache Kafka)", fontsize=8.5, va='center', color='#212121')

    # DB note
    ax.text(12.6, 0.55, "У каждого сервиса своя БД!", fontsize=8.5, va='center', fontweight='bold', color='#B71C1C')

    plt.tight_layout()
    plt.savefig('docs/architecture.png', dpi=200, bbox_inches='tight')
    plt.close()
    print('Generated docs/architecture.png')

render_architecture_png()

# -----------------------------------------------------------------------------
# 4. Рендеринг PNG: docs/sequence-success.png (Happy Path)
# -----------------------------------------------------------------------------
def render_sequence_success_png():
    fig, ax = plt.subplots(figsize=(16, 12), dpi=200)
    ax.set_xlim(0, 16)
    ax.set_ylim(0, 16)
    ax.axis('off')

    ax.text(8, 15.6, "Диаграмма последовательности: Успешная покупка билета в цирк (Happy Path)", 
            ha='center', va='center', fontsize=16, fontweight='bold', color='#1A237E')
    ax.text(8, 15.2, "Сценарий: бронирование мест + списание баллов лояльности + оплата картой + отправка электронных билетов", 
            ha='center', va='center', fontsize=10.5, color='#546E7A')

    # Participants
    participants = [
        ("Клиент\n(Web/App)", 1.2, '#E1F5FE', '#0288D1'),
        ("API Gateway", 3.0, '#BBDEFB', '#1976D2'),
        ("booking\nservice", 4.9, '#FFF3E0', '#F57C00'),
        ("catalog\nservice", 6.8, '#E8F5E9', '#388E3C'),
        ("loyalty\nservice", 8.7, '#F3E5F5', '#7B1FA2'),
        ("payment\nservice", 10.6, '#E0F2F1', '#00796B'),
        ("Платёжный\nшлюз банка", 12.5, '#EFEBE9', '#5D4037'),
        ("Apache\nKafka", 14.1, '#ECEFF1', '#455A64'),
        ("notification\nservice", 15.3, '#FCE4EC', '#C2185B'),
    ]

    # Draw lifeline headers and vertical lines
    for name, x, bg, border in participants:
        rect = patches.FancyBboxPatch((x-0.7, 14.1), 1.4, 0.8, boxstyle="round,pad=0.05,rounding_size=0.1",
                                      facecolor=bg, edgecolor=border, linewidth=1.5)
        ax.add_patch(rect)
        ax.text(x, 14.5, name, ha='center', va='center', fontsize=8.5, fontweight='bold')
        ax.plot([x, x], [0.6, 14.1], color='#B0BEC5', lw=1, ls=':')

    # Helper for message arrows
    def msg(y, x1, x2, text, is_async=False, is_response=False, color='#263238'):
        ls = '--' if (is_async or is_response) else '-'
        ax.annotate("", xy=(x2, y), xytext=(x1, y),
                    arrowprops=dict(arrowstyle="->", color=color, lw=1.5, ls=ls))
        mid = (x1 + x2) / 2
        va = 'bottom'
        ax.text(mid, y + 0.08, text, ha='center', va=va, fontsize=8, color=color, fontweight='bold' if not is_response else 'normal')

    # Step by step execution
    # 1. Client creates booking
    msg(13.6, 1.2, 3.0, "1. POST /api/v1/bookings (сеанс 105, ряд 3 место 12, списать 300 баллов, Idempotency-Key)", color='#0D47A1')
    msg(13.2, 3.0, 4.9, "2. Проксирование (валидация JWT токена)", color='#0D47A1')

    # 3. Booking validates session in catalog
    msg(12.7, 4.9, 6.8, "3. GET /internal/v1/sessions/105", color='#1B5E20')
    msg(12.3, 6.8, 4.9, "4. 200 OK (сеанс активен, цена 2500 руб.)", is_response=True, color='#1B5E20')

    # 5. Booking locks seats in Redis
    rect_redis = patches.Rectangle((4.7, 11.6), 0.4, 0.45, facecolor='#FFCC80', edgecolor='#E65100')
    ax.add_patch(rect_redis)
    ax.text(5.2, 11.8, "5. Redis: SETNX hold:105:3_12 (TTL 15 мин)", fontsize=7.5, color='#E65100', fontweight='bold')

    # 6. Booking reserves points in loyalty
    msg(11.2, 4.9, 8.7, "6. POST /internal/v1/loyalty/reservations (заморозить 300 баллов)", color='#4A148C')
    msg(10.8, 8.7, 4.9, "7. 201 Created (reservationId, статус RESERVED)", is_response=True, color='#4A148C')

    # 8. Booking creates payment in payment-service
    msg(10.3, 4.9, 10.6, "8. POST /internal/v1/payments (сумма: 2200 руб. = 2500 - 300)", color='#004D40')
    msg(9.9, 10.6, 12.5, "9. Инициализация сессии эквайринга", color='#3E2723')
    msg(9.5, 12.5, 10.6, "10. URL платёжной формы банка", is_response=True, color='#3E2723')
    msg(9.1, 10.6, 4.9, "11. 201 Created (paymentUrl, status: PENDING)", is_response=True, color='#004D40')

    # 12. Booking returns paymentUrl to Client
    msg(8.6, 4.9, 3.0, "12. 201 Created (Booking: PENDING_PAYMENT, expiresAt=15 мин, paymentUrl)", is_response=True, color='#E65100')
    msg(8.2, 3.0, 1.2, "13. Редирект на платёжную форму", is_response=True, color='#0D47A1')

    # 14. Client pays at bank
    msg(7.6, 1.2, 12.5, "14. Зритель вводит данные карты и подтверждает оплату 2200 руб.", color='#C62828')

    # 15. Bank sends webhook to payment-service
    msg(7.0, 12.5, 10.6, "15. Webhook: Оплата успешна (HMAC X-Signature)", color='#2E7D32')

    # 16. payment-service publishes payment.completed to Kafka
    msg(6.4, 10.6, 14.1, "16. Публикация события: payment.completed (топик payments, key=bookingId)", is_async=True, color='#00796B')

    # 17. booking-service consumes payment.completed
    msg(5.8, 14.1, 4.9, "17. Консьюминг: payment.completed", is_async=True, color='#F57C00')
    
    # Booking updates DB
    rect_db = patches.Rectangle((4.7, 5.0), 0.4, 0.5, facecolor='#FFE082', edgecolor='#FF8F00')
    ax.add_patch(rect_db)
    ax.text(5.2, 5.25, "18. Запись билетов в PostgreSQL, DEL из Redis, статус CONFIRMED", fontsize=7.5, color='#E65100', fontweight='bold')

    # 19. booking publishes booking.confirmed to Kafka
    msg(4.6, 4.9, 14.1, "19. Публикация события: booking.confirmed (билеты выпущены)", is_async=True, color='#E65100')

    # 20. loyalty-service consumes booking.confirmed
    msg(4.0, 14.1, 8.7, "20. Консьюминг: booking.confirmed", is_async=True, color='#7B1FA2')
    ax.text(8.7, 3.65, "21. Финализация списания 300 баллов + начисление 220 бонусов кэшбэка", fontsize=7.5, ha='center', color='#4A148C', fontweight='bold')

    # 22. notification-service consumes booking.confirmed
    msg(3.2, 14.1, 15.3, "22. Консьюминг: booking.confirmed", is_async=True, color='#C2185B')
    msg(2.6, 15.3, 1.2, "23. Отправка Email и SMS с PDF-билетом и QR-кодом для прохода на манеж", is_async=True, color='#880E4F')

    # Result state
    res_box = patches.FancyBboxPatch((1.2, 1.0), 14.1, 0.8, boxstyle="round,pad=0.05,rounding_size=0.1",
                                     facecolor='#E8F5E9', edgecolor='#2E7D32', linewidth=1.5)
    ax.add_patch(res_box)
    ax.text(8.25, 1.4, "Итог: Все сервисы консистентны. Места проданы, баллы начислены, билеты у клиента. Сага завершена успешно!", 
            ha='center', va='center', fontsize=9.5, fontweight='bold', color='#1B5E20')

    plt.tight_layout()
    plt.savefig('docs/sequence-success.png', dpi=200, bbox_inches='tight')
    plt.close()
    print('Generated docs/sequence-success.png')

render_sequence_success_png()

# -----------------------------------------------------------------------------
# 5. Рендеринг PNG: docs/sequence-failure.png (Сбой и сага-компенсации)
# -----------------------------------------------------------------------------
def render_sequence_failure_png():
    fig, ax = plt.subplots(figsize=(16, 12), dpi=200)
    ax.set_xlim(0, 16)
    ax.set_ylim(0, 16)
    ax.axis('off')

    ax.text(8, 15.6, "Диаграмма последовательности: Сбои и компенсирующие действия (Паттерн Сага)", 
            ha='center', va='center', fontsize=16, fontweight='bold', color='#B71C1C')
    ax.text(8, 15.2, "Сценарии: А) Отклонение оплаты банком • Б) Истечение 15-минутного таймера • В) Запоздалая оплата", 
            ha='center', va='center', fontsize=10.5, color='#546E7A')

    # Participants
    participants = [
        ("Клиент\n(Web/App)", 1.2, '#E1F5FE', '#0288D1'),
        ("booking\nservice", 3.8, '#FFF3E0', '#F57C00'),
        ("Redis\n(Locks)", 6.0, '#FFECB3', '#FFA000'),
        ("loyalty\nservice", 8.2, '#F3E5F5', '#7B1FA2'),
        ("payment\nservice", 10.4, '#E0F2F1', '#00796B'),
        ("Банк / Шлюз", 12.4, '#EFEBE9', '#5D4037'),
        ("Apache\nKafka", 14.1, '#ECEFF1', '#455A64'),
        ("notification\nservice", 15.3, '#FCE4EC', '#C2185B'),
    ]

    for name, x, bg, border in participants:
        rect = patches.FancyBboxPatch((x-0.7, 14.1), 1.4, 0.8, boxstyle="round,pad=0.05,rounding_size=0.1",
                                      facecolor=bg, edgecolor=border, linewidth=1.5)
        ax.add_patch(rect)
        ax.text(x, 14.5, name, ha='center', va='center', fontsize=8.5, fontweight='bold')
        ax.plot([x, x], [0.6, 14.1], color='#B0BEC5', lw=1, ls=':')

    def msg(y, x1, x2, text, is_async=False, is_response=False, color='#263238'):
        ls = '--' if (is_async or is_response) else '-'
        ax.annotate("", xy=(x2, y), xytext=(x1, y),
                    arrowprops=dict(arrowstyle="->", color=color, lw=1.5, ls=ls))
        mid = (x1 + x2) / 2
        ax.text(mid, y + 0.08, text, ha='center', va='bottom', fontsize=8, color=color, fontweight='bold' if not is_response else 'normal')

    # Section A: Payment Failed
    secA = patches.Rectangle((0.6, 9.8), 15.0, 4.0, facecolor='#FFEBEE', edgecolor='#EF5350', alpha=0.5, linestyle=':')
    ax.add_patch(secA)
    ax.text(0.8, 13.5, "СЦЕНАРИЙ А: Отказ в оплате банком (Недостаточно средств на карте)", fontsize=9, fontweight='bold', color='#C62828')

    msg(13.1, 12.4, 10.4, "1. Webhook банка: Оплата отклонена (FAIL: Insufficient funds)", color='#C62828')
    msg(12.5, 10.4, 14.1, "2. Публикация: payment.failed (key=bookingId)", is_async=True, color='#D32F2F')
    msg(11.9, 14.1, 3.8, "3. Консьюминг: payment.failed", is_async=True, color='#E65100')
    msg(11.3, 3.8, 6.0, "4. Компенсация: DEL hold:105:3_12 (Снятие блокировки места)", color='#E65100')
    msg(10.7, 3.8, 8.2, "5. Компенсация: DELETE /reservations/{id} (Разморозка баллов)", color='#6A1B9A')
    msg(10.1, 3.8, 1.2, "6. Уведомление клиента: Оплата не прошла, бронь аннулирована, баллы возвращены", color='#C62828')

    # Section B: Expiration timeout (15 min)
    secB = patches.Rectangle((0.6, 5.6), 15.0, 3.9, facecolor='#FFF8E1', edgecolor='#FFA726', alpha=0.5, linestyle=':')
    ax.add_patch(secB)
    ax.text(0.8, 9.2, "СЦЕНАРИЙ Б: Истечение 15 минут без оплаты (Таймаут удержания)", fontsize=9, fontweight='bold', color='#E65100')

    ax.text(3.8, 8.8, "1. Фоновый воркер booking-service обнаружил expiresAt < now", fontsize=8, color='#E65100', fontweight='bold')
    msg(8.4, 3.8, 6.0, "2. Снятие удержания: TTL истек / DEL ключа в Redis", color='#E65100')
    msg(7.8, 3.8, 8.2, "3. Компенсация: Разморозка зарезервированных баллов", color='#6A1B9A')
    msg(7.2, 3.8, 14.1, "4. Публикация: booking.expired (места возвращены в свободную продажу)", is_async=True, color='#E65100')
    msg(6.6, 14.1, 15.3, "5. Консьюминг: booking.expired", is_async=True, color='#C2185B')
    msg(6.0, 15.3, 1.2, "6. Email/SMS: 'Ваше время бронирования истекло, места освобождены'", is_async=True, color='#880E4F')

    # Section C: Late payment arrives after expiry
    secC = patches.Rectangle((0.6, 1.5), 15.0, 3.8, facecolor='#EDE7F6', edgecolor='#7E57C2', alpha=0.5, linestyle=':')
    ax.add_patch(secC)
    ax.text(0.8, 5.0, "СЦЕНАРИЙ В: Запоздалая оплата после истечения брони (Компенсирующий возврат 100%)", fontsize=9, fontweight='bold', color='#4527A0')

    msg(4.6, 12.4, 10.4, "1. Запоздалый Webhook успешной оплаты от банка", color='#2E7D32')
    msg(4.1, 10.4, 14.1, "2. payment.completed доставлен в Kafka", is_async=True, color='#00796B')
    msg(3.6, 14.1, 3.8, "3. booking-service видит статус брони = EXPIRED (место уже мог купить другой!)", is_async=True, color='#B71C1C')
    msg(3.0, 3.8, 10.4, "4. Сага-компенсация: POST /internal/v1/payments/{id}/refund (возврат 100% денег на карту)", color='#B71C1C')
    msg(2.4, 10.4, 12.4, "5. Вызов эквайринга: Возврат 2200 руб. на карту плательщика", color='#3E2723')
    msg(1.8, 10.4, 15.3, "6. Уведомление: 'Сеанс брони истек. Деньги успешно возвращены на вашу карту'", color='#880E4F')

    # Result state
    res_box = patches.FancyBboxPatch((1.2, 0.6), 14.1, 0.7, boxstyle="round,pad=0.05,rounding_size=0.1",
                                     facecolor='#FAFAFA', edgecolor='#455A64', linewidth=1.5)
    ax.add_patch(res_box)
    ax.text(8.25, 0.95, "Гарантия согласованности: Никаких 'зависших' денег или заблокированных мест. Паттерн Сага гарантирует Eventual Consistency.", 
            ha='center', va='center', fontsize=9, fontweight='bold', color='#263238')

    plt.tight_layout()
    plt.savefig('docs/sequence-failure.png', dpi=200, bbox_inches='tight')
    plt.close()
    print('Generated docs/sequence-failure.png')

render_sequence_failure_png()
print('ALL DIAGRAMS SUCCESSFULLY GENERATED!')
