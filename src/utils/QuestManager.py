from typing import Optional, Dict, Any
from src.database.quest import get_user_quests, get_quest_progress, update_quest_progress, complete_quest as db_complete_quest, start_quest
from src.models.Quest import QuestRegistry, QuestStepType, QuestReward
from src.database.balance import update_balance
from src.database.achievement import check_and_unlock_achievements, increment_stat
from src.database.job import add_job_xp

class QuestManager:
    @staticmethod
    def on_activity(user_id: int, activity: str, amount: int = 1):
        active_quests = get_user_quests(user_id, status='ACTIVE')
        for q_data in active_quests:
            quest_id = q_data['quest_id']
            quest = QuestRegistry.get_quest(quest_id)
            if not quest:
                continue

            progress = get_quest_progress(user_id, quest_id)
            if not progress:
                continue

            step_idx = progress['step_index']
            if step_idx >= len(quest.steps):
                continue

            step = quest.steps[step_idx]
            if step.step_type != QuestStepType.ACTIVITY:
                continue

            if quest.on_activity(user_id, activity, amount, progress):
                QuestManager.advance_step(user_id, quest_id)

    @staticmethod
    def advance_step(user_id: int, quest_id: str, choice_id: Optional[str] = None):
        quest = QuestRegistry.get_quest(quest_id)
        if not quest:
            return

        progress = get_quest_progress(user_id, quest_id)
        if not progress:
            return

        step_idx = progress['step_index']
        if step_idx >= len(quest.steps):
            return

        current_step = quest.steps[step_idx]
        
        if current_step.rewards:
            QuestManager.award_rewards(user_id, current_step.rewards)

        if current_step.step_type == QuestStepType.CHOICE and choice_id:
            next_idx = quest.on_choice(user_id, choice_id, progress)
        else:
            next_idx = step_idx + 1

        if next_idx >= len(quest.steps):
            QuestManager.complete_quest(user_id, quest_id)
        else:
            update_quest_progress(user_id, quest_id, step_index=next_idx, progress_value=0)

    @staticmethod
    def complete_quest(user_id: int, quest_id: str):
        quest = QuestRegistry.get_quest(quest_id)
        if not quest:
            return

        db_complete_quest(user_id, quest_id)

        for next_q_id in quest.get_unlocks():
            start_quest(user_id, next_q_id)

    @staticmethod
    def award_rewards(user_id: int, rewards: QuestReward):
        if rewards.money > 0:
            update_balance(user_id, rewards.money)
        
        if rewards.xp > 0:
            add_job_xp(user_id, "adventurer", rewards.xp)
            
        if rewards.achievement_id:
            from src.database.achievement import get_connection
            conn = get_connection()
            try:
                conn.execute(
                    "INSERT OR IGNORE INTO user_achievements (user_id, achievement_id) VALUES (?, ?)",
                    (user_id, rewards.achievement_id)
                )
                conn.commit()
            finally:
                conn.close()
        
        if rewards.items:
            from src.database.item import add_item_to_inventory
            for item in rewards.items:
                add_item_to_inventory(user_id, item, 1)
