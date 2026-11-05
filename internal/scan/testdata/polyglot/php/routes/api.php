<?php
use Illuminate\Support\Facades\Route;
Route::get('/invoices/{id}', [InvoiceController::class, 'show']);
