export interface Category {
  id: string;
  name: string;
  description?: string;
  color?: string;
  parent_id?: string;
  subcategories?: Category[];
}

export interface Person {
  id: string;
  name: string;
  email?: string;
  created_at: string;
  updated_at: string;
}

export interface Transaction {
  id: string;
  description: string;
  amount: number;
  assigned_to: string[];
  date_uploaded: string;
  file_name?: string;
  transaction_date?: string;
  posted_date?: string;
  card_number?: string;
  category_id?: string | null;
  splits?: TransactionSplit[];
}

export interface TransactionSplit {
  id?: string;
  transaction_id?: string;
  amount: number;
  category_id: string;
  notes?: string;
}

export interface Archive {
  id: string;
  description?: string;
  archived_at: string;
  transaction_count: number;
  total_amount: number;
  person_totals?: PersonTotal[];
  created_at: string;
  updated_at: string;
}

export interface PersonTotal {
  person: string;
  total: number;
}

export interface Rule {
  id: string;
  match_value: string;
  category_id: string;
  category_name: string;
  priority: number;
  created_at: string;
  updated_at: string;
}
